package discover

import (
    "context"
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/ethereum/go-ethereum/log"
    "github.com/ipfs/interface-go-ipfs-core"

    "github.com/Tau-Coin/taucoin-go-p2p/ipfs/api"
    "github.com/Tau-Coin/taucoin-go-p2p/p2p/tnode"
)

var (
    ErrNilAPI = errors.New("nil ipfs api")

    ErrNilDB  = errors.New("nil node database")
)

const (
    defDiscoverInterval = 10 * time.Second
)

type Discover struct {

    ctx    context.Context

    // ensure running once for discover routine
    runner sync.Once

    // node database
    db     *tnode.DB

    // chan for new connected nodes
    Conn   chan *tnode.Node

    // chan for new disconnected nodes
    Disc   chan *tnode.Node

    // chan for quit
    quit   chan struct{}

    // ipfs api
    ipfs   *api.API

    // ipfs connected peers filter
    filter Filter

    log    log.Logger

    wg     sync.WaitGroup
}

func New(ctx context.Context, api *api.API, db *tnode.DB, log log.Logger) (*Discover, error) {

    if api == nil {
        return nil, ErrNilAPI
    }
    if db == nil {
        return nil, ErrNilDB
    }

    return &Discover{
        ctx:    ctx,
        db:     db,
        Conn:   make(chan *tnode.Node),
        Disc:   make(chan *tnode.Node),
        quit:   make(chan struct{}, 1),
        ipfs:   api,
        filter: newFilter(),
        log:    log,
    }, nil
}

func (d *Discover) Start() {
    go d.runner.Do(d.discover)
}

func (d *Discover) Stop() {
    d.quit <-struct{}{}
    d.wg.Wait()
}

func (d *Discover) discover() {

    d.wg.Add(1)
    defer d.wg.Done()
    d.log.Info("Starting discover")
    defer d.log.Info("Discover stopped")

    ticker := time.NewTicker(defDiscoverInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Anyway, we should get local ipfs id
            if d.db.Home() == nil {
                id, err := d.ipfs.HttpAPI().Key().Self(d.ctx)
                // ipfs daemon hasn't been launched.
                if err != nil {
                    continue
                }

                home := tnode.NewNode(string(id.ID()), nil, nil)
                d.log.Info("Home node got", "id", id.ID())
                d.db.SetHome(home)
            }

            // get swarm peers.
            // Note: non-nil error means ipfs daemon maybe dead
            // and empty peers mean network disconnected.
            // In this condition, disconnect all active peers.
            var peers []iface.ConnectionInfo
            peers, _ = d.ipfs.HttpAPI().Swarm().Peers(d.ctx)
            if len(peers) == 0 && d.db.Size() == 0 {
                d.log.Warn("Got nil peers and try again")
                continue
            }

            d.log.Debug("raw peers queried", "number", len(peers))
            for _, p := range peers {
                d.log.Debug("raw peer queried", "peer", fmt.Sprintf("%s/%s", p.Address(), p.ID()))
            }

            wanted := d.filter.Filter(peers)
            for _, w := range wanted {
                d.log.Debug("wanted peer", "peer", fmt.Sprintf("%s/%s", w.Address(), w.ID()))
            }
            latest := tnode.MakeNodesMap(wanted)

            // decide which nodes are connected newly and
            // which nodes should be disconnected.
            connected, disconnected := d.db.Choose(latest)
            d.db.Replace(latest)

            // notify
            // first of all, notify nodes which should be disconnected
            for _, disc := range disconnected {
                d.Disc <- disc
            }

            for _, con := range connected {
                d.Conn <- con
            }

        case <-d.quit:
            return
        }
    }
}
