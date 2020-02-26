package tau

import (
    "errors"
    "fmt"
    "sync"

    "github.com/Tau-Coin/taucoin-go-p2p/p2p"
)

var (
    errClosed            = errors.New("peer set is closed")
    errAlreadyRegistered = errors.New("peer is already registered")
    errNotRegistered     = errors.New("peer is not registered")
)

type peer struct {
    id string

    *p2p.Peer
    rw p2p.MsgReadWriter

    version int
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
    return &peer {
            Peer:    p,
            rw:      rw,
            id:      fmt.Sprintf("%x", p.ID().Bytes()[:8]),
            version: version,
    }
}

// The following methods will be customized according to the real requirement.
// e.g.
// func (p *peer) publishBlockList() {}
// func (p *peer) publishTxList() {}

func (p *peer) close() {
}

// String implements fmt.Stringer.
func (p *peer) String() string {
    return fmt.Sprintf("Peer %s [%s]", p.id,
        fmt.Sprintf("eth/%2d", p.version),
    )
}

// peerSet represents the collection of active peers currently participating in
// the TAU sub-protocol.
type peerSet struct {
        peers  map[string]*peer
        lock   sync.RWMutex
        closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
    return &peerSet{
        peers: make(map[string]*peer),
    }
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
    ps.lock.Lock()
    defer ps.lock.Unlock()

    if ps.closed {
        return errClosed
    }
    if _, ok := ps.peers[p.id]; ok {
        return errAlreadyRegistered
    }
    ps.peers[p.id] = p

    //go p.broadcastBlocks()
    //go p.broadcastTransactions()
    //go p.announceTransactions()

    return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
    ps.lock.Lock()
    defer ps.lock.Unlock()

    p, ok := ps.peers[id]
    if !ok {
        return errNotRegistered
    }
    delete(ps.peers, id)
    p.close()

    return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
    ps.lock.RLock()
    defer ps.lock.RUnlock()

    return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
    ps.lock.RLock()
    defer ps.lock.RUnlock()

    return len(ps.peers)
}


// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
    ps.lock.Lock()
    defer ps.lock.Unlock()

    for _, p := range ps.peers {
        p.Disconnect(p2p.DiscQuitting)
    }
    ps.closed = true
}
