package p2p

import (
    "context"

    "github.com/ipfs/interface-go-ipfs-core"
    caopts "github.com/ipfs/interface-go-ipfs-core/options"

    "github.com/Tau-Coin/taucoin-go-p2p/ipfs/api"
)

type Publisher interface {
    Pub(string, []byte) error
}

type Subscriber interface {
    Sub(string) (iface.PubSubSubscription, error)
}

type PublishSubscriber interface {
    Publisher
    Subscriber
}

var _ PublishSubscriber = &pubsub{}

type pubsub struct {
    ctx  context.Context

    ipfs *api.API
}

func newpubsub(ctx context.Context, ipfs *api.API) *pubsub {
    return &pubsub{
            ctx:  ctx,
            ipfs: ipfs,
    }
}

func (ps *pubsub) Pub(topic string, payload []byte) error {
    return ps.ipfs.HttpAPI().PubSub().Publish(ps.ctx, topic, payload)
}

func (ps *pubsub) Sub(topic string) (iface.PubSubSubscription, error) {
    return ps.ipfs.HttpAPI().PubSub().Subscribe(ps.ctx, topic, caopts.PubSub.Discover(true))
}
