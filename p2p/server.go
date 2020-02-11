package p2p

import (
    "context"
    "errors"
    "sync"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/common/mclock"
    "github.com/ethereum/go-ethereum/event"
    "github.com/ethereum/go-ethereum/log"

    "github.com/Tau-Coin/taucoin-go-p2p/p2p/discover"
    ipfsapi"github.com/Tau-Coin/taucoin-go-p2p/ipfs/api"
    "github.com/Tau-Coin/taucoin-go-p2p/p2p/tnode"
)

var errServerStopped = errors.New("server stopped")

// Config holds Server options.
type Config struct {

    Ctx context.Context

    // MaxPeers is the maximum number of peers that can be
    // subscribed. It must be greater than zero.
    MaxPeers int

    // Protocols should contain the protocols supported
    // by the server. Matching protocols are launched for
    // each peer.
    Protocols []Protocol `toml:"-"`

    // If EnableMsgEvents is set then the server will emit PeerEvents
    // whenever a message is sent to or received from a peer
    EnableMsgEvents bool

    // Logger is a custom logger to use with the p2p.Server.
    Logger log.Logger `toml:",omitempty"`
}

// Server manages all peer which are subscribed to ipfs daemon.
type Server struct {
    // Config fields may not be modified while the server is running.
    Config

    lock    sync.Mutex // protects running
    running bool

    ctx      context.Context
    loopWG   sync.WaitGroup // loop
    peerFeed event.Feed
    log      log.Logger

    nodedb    *tnode.DB
    homenode  *tnode.Node
    discover  *discover.Discover

    ipfs   *ipfsapi.API
    pubsub PublishSubscriber

    // Channels into the run loop.
    quit              chan struct{}
    peerOp            chan peerOpFunc
    peerOpDone        chan struct{}
    delpeer           chan peerDrop
    checkpointAddPeer chan *tnode.Node
}

type peerOpFunc func(map[tnode.ID]*Peer)

type peerDrop struct {
    *Peer
    err       error
    requested bool // true if signaled by the peer
}

// Stop terminates the server and all active peer subscription.
// It blocks until all active subscriptions have been closed.
func (srv *Server) Stop() {
    srv.log.Info("Stopping p2p server")
    srv.lock.Lock()

    if !srv.running {
        srv.lock.Unlock()
        return
    }
    srv.running = false
    close(srv.quit)

    srv.lock.Unlock()
    srv.loopWG.Wait()
}

// Start starts running the server.
// Servers can not be re-used after stopping.
func (srv *Server) Start() (err error) {
    srv.lock.Lock()
    defer srv.lock.Unlock()
    if srv.running {
        return errors.New("server already running")
    }
    srv.running = true

    srv.log = srv.Config.Logger
    if srv.log == nil {
        srv.log = log.Root()
    }
    srv.log.Info("Starting p2p server")
    srv.ctx = srv.Config.Ctx
    if srv.ctx == nil {
        srv.ctx = context.Background()
    }

    srv.quit = make(chan struct{})
    srv.delpeer = make(chan peerDrop)
    srv.checkpointAddPeer = make(chan *tnode.Node)
    srv.peerOp = make(chan peerOpFunc)
    srv.peerOpDone = make(chan struct{})

    if err := srv.setupIpfsAPI(); err != nil {
        return err
    }
    if err := srv.setupDiscovery(); err != nil {
        return err
    }

    srv.loopWG.Add(1)
    go srv.run()

    return nil
}

// Peers returns all connected peers.
func (srv *Server) Peers() []*Peer {
    var ps []*Peer
    select {
    // Note: We'd love to put this function into a variable but
    // that seems to cause a weird compiler error in some
    // environments.
    case srv.peerOp <- func(peers map[tnode.ID]*Peer) {
        for _, p := range peers {
            ps = append(ps, p)
        }
    }:
        <-srv.peerOpDone
    case <-srv.quit:
    }
    return ps
}

// PeerCount returns the number of connected peers.
func (srv *Server) PeerCount() int {
    var count int
    select {
    case srv.peerOp <- func(ps map[tnode.ID]*Peer) { count = len(ps) }:
        <-srv.peerOpDone
    case <-srv.quit:
    }
    return count
}

func (srv *Server) IPFS() *ipfsapi.API {
    return srv.ipfs
}

func (srv *Server) setupIpfsAPI() error {
    var err error

    if srv.ipfs, err = ipfsapi.New(nil); err != nil {
        return err
    }

    srv.pubsub = newpubsub(srv.ctx, srv.ipfs)
    return nil
}

func (srv *Server) setupDiscovery() error {
    var err error

    srv.nodedb = tnode.NewDB(0)
    srv.discover, err = discover.New(srv.ctx, srv.ipfs, srv.nodedb, srv.log)

    srv.discover.Start()

    return err
}

func (srv *Server) run() {
    srv.log.Info("Started P2P networking")
    defer srv.loopWG.Done()
    defer srv.discover.Stop()
    defer srv.nodedb.Close()

    var (
        peers = make(map[tnode.ID]*Peer)
    )

running:
    for {
        select {
        case <-srv.quit:
            // The server was stopped. Run the cleanup logic.
            break running

        case op := <-srv.peerOp:
            // This channel is used by Peers and PeerCount.
            op(peers)
            srv.peerOpDone <- struct{}{}

        case n := <-srv.discover.Conn:
            srv.log.Debug("Ipfs new romote node connected", "node", n)

            // Get home node when connected node arrives.
            if srv.homenode == nil {
                srv.homenode = srv.nodedb.Home()
            }

            if len(peers) >= srv.Config.MaxPeers {
                srv.log.Debug("Ignore new romote node", "peers", len(peers), "max", srv.Config.MaxPeers)
            } else if _, ok := peers[n.ID()]; !ok {
               // run peer
               srv.checkpointAddPeer <- n
            }

        case n := <-srv.discover.Disc:
            srv.log.Debug("Ipfs romote node disconnected", "node", n)

            if p, ok := peers[n.ID()]; ok {
                p.Disconnect(DiscNetworkError)
            }

        case n := <-srv.checkpointAddPeer:
            err := srv.addPeerChecks(peers, n)
            if err == nil {
                p := newPeer(srv.log, n, string(srv.homenode.ID()), srv.Protocols, srv.pubsub)
                if srv.EnableMsgEvents {
                    p.events = &srv.peerFeed
                }

                go srv.runPeer(p)
                peers[n.ID()] = p
            }

        case pd := <-srv.delpeer:
            // A peer disconnected.
            d := common.PrettyDuration(mclock.Now() - pd.created)
            pd.log.Debug("Removing p2p peer", "addr", pd.RemoteAddr(), "peers", len(peers)-1, "duration", d, "req", pd.requested, "err", pd.err)
            delete(peers, pd.ID())
        }
    }

    srv.log.Trace("P2P networking is spinning down")

    // Disconnect all peers.
    for _, p := range peers {
        p.Disconnect(DiscQuitting)
    }
    // Wait for peers to shut down.
    for len(peers) > 0 {
        p := <-srv.delpeer
        p.log.Trace("<-delpeer ", "id", p.ID())
        delete(peers, p.ID())
    }
}

func (srv *Server) addPeerChecks(_ map[tnode.ID]*Peer, _ *tnode.Node) error {
    return nil
}

// runPeer runs in its own goroutine for each peer.
// it waits until the Peer logic returns and removes
// the peer.
func (srv *Server) runPeer(p *Peer) {

    // broadcast peer add
    srv.peerFeed.Send(&PeerEvent{
        Type:          PeerEventTypeAdd,
        Peer:          p.ID(),
        RemoteAddress: p.RemoteAddr(),
        LocalAddress:  string(srv.homenode.ID()),
    })

    // run the protocol
    remoteRequested, err := p.run()

    // broadcast peer drop
    srv.peerFeed.Send(&PeerEvent{
        Type:          PeerEventTypeDrop,
        Peer:          p.ID(),
        Error:         err.Error(),
        RemoteAddress: p.RemoteAddr(),
        LocalAddress:  string(srv.homenode.ID()),
    })

    // Note: run waits for existing peers to be sent on srv.delpeer
    // before returning, so this send should not select on srv.quit.
    srv.delpeer <- peerDrop{p, err, remoteRequested}
}
