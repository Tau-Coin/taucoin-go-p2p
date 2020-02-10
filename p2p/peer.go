package p2p

import (
    "bytes"
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "sync"
    "strings"
    "time"

    "github.com/ethereum/go-ethereum/common/mclock"
    "github.com/ethereum/go-ethereum/event"
    "github.com/ethereum/go-ethereum/log"

    "github.com/ipfs/interface-go-ipfs-core"

    "github.com/Tau-Coin/taucoin-go-p2p/p2p/tnode"
)

var (
    ErrShuttingDown = errors.New("shutting down")

    ErrReadTimeout  = errors.New("read timeout")
)

var (
    readTimeoutThreshold = 120 * time.Second
)

// PeerEventType is the type of peer events emitted by a p2p.Server
type PeerEventType string

const (
	// PeerEventTypeAdd is the type of event emitted when a peer is added
	// to a p2p.Server
	PeerEventTypeAdd PeerEventType = "add"

	// PeerEventTypeDrop is the type of event emitted when a peer is
	// dropped from a p2p.Server
	PeerEventTypeDrop PeerEventType = "drop"

	// PeerEventTypeMsgSend is the type of event emitted when a
	// message is successfully sent to a peer
	PeerEventTypeMsgSend PeerEventType = "msgsend"

	// PeerEventTypeMsgRecv is the type of event emitted when a
	// message is received from a peer
	PeerEventTypeMsgRecv PeerEventType = "msgrecv"
)

// PeerEvent is an event emitted when peers are either added or dropped from
// a p2p.Server or when a message is sent or received on a peer connection
type PeerEvent struct {
	Type          PeerEventType `json:"type"`
	Peer          tnode.ID      `json:"peer"`
	Error         string        `json:"error,omitempty"`
	Protocol      string        `json:"protocol,omitempty"`
	LocalAddress  string        `json:"local,omitempty"`
	RemoteAddress string        `json:"remote,omitempty"`
}

// Peer represents a connected remote node.
type Peer struct {
    node   *tnode.Node

    pubsub  PublishSubscriber
    running map[string]*protoRW
    log     log.Logger
    created mclock.AbsTime

    wg       sync.WaitGroup
    protoErr chan error
    closed   chan struct{}
    disc     chan DiscReason

    // events receives message send / receive events if set
    events *event.Feed
}

func (p *Peer) Disconnect(reason DiscReason) {
    select {
    case p.disc <- reason:
    case <-p.closed:
    }
}

func newPeer(log log.Logger, node *tnode.Node, homeID string, protocols []Protocol, pubsub PublishSubscriber) *Peer {
    protomap := matchProtocols(protocols, pubsub, homeID)
    p := &Peer{
        node:     node,
        pubsub:   pubsub,
        running:  protomap,
        created:  mclock.Now(),
        disc:     make(chan DiscReason),
        protoErr: make(chan error, len(protomap)),
        closed:   make(chan struct{}),
        log:      log.New("id", node.ID()),
    }

    return p
}

func (p *Peer) ID() tnode.ID {
    return p.node.ID()
}

func (p *Peer) RemoteAddr() string {
    return p.node.Addr().String()
}

func (p *Peer) run() (remoteRequested bool, err error) {
    var (
        writeStart = make(chan struct{}, 1)
        writeErr   = make(chan error, 1)
        readErr    = make(chan error, 1) // TODO:p.readLoopCount()
        //reason     DiscReason // sent to the peer
        subChans   = make(map[string]iface.PubSubSubscription)
    )

    p.wg.Add(p.readLoopCount())
    for _, proto := range p.running {
        for _, t := range proto.Topics {
            topic := constructTopic(string(p.ID()), t)
            go p.readLoop(topic, readErr, subChans)
        }
    }

    // Start all protocol handlers.
    writeStart <- struct{}{}
    p.startProtocols(writeStart, writeErr)

    // Wait for an error or disconnect.
loop:
    for {
        select {
        case err = <-writeErr:
            // A write finished. Allow the next write to start if
            // there was no error.
            if err != nil {
                //reason = DiscNetworkError
                break loop
            }
            writeStart <- struct{}{}

        case err = <-readErr:
            if _, ok := err.(DiscReason); ok {
                remoteRequested = true
                //reason = r
            }
            break loop

        case err = <-p.protoErr:
            //reason = discReasonForError(err)
            if err == ErrReadTimeout {
                p.log.Trace(fmt.Sprintf("Peer %s timeout", p.ID()))
            }
            break loop

        case err = <-p.disc:
            //reason = discReasonForError(err)
            break loop
        }
    }

    // Close all active subscription channels
    for _, sub := range subChans {
        sub.Close()
    }

    close(p.closed)
    p.wg.Wait()

    return remoteRequested, err
}

func (p *Peer) readLoop(topic string, errc chan<- error, subChans map[string]iface.PubSubSubscription) {
    defer p.wg.Done()

    subscription, err := p.pubsub.Sub(topic)
    if err != nil {
        errc <- err
        return
    }
    subChans[topic] = subscription
    defer delete(subChans, topic)

    ps := p.pubsub.(*pubsub)

    for {
        message, err := subscription.Next(ps.ctx)
        if err != nil {
            errc <- err
            return
        }

        if err = p.handle(message, topic); err != nil {
            errc <- err
            return
        }
    }
}

func (p *Peer) handle(message iface.PubSubMessage, topic string) error {
    msg := Msg{
        From:       string(message.From()),
        Topic:      topic,
        Payload:    bytes.NewReader(message.Data()),
        ReceivedAt: time.Now(),
    }

    // select handling protocol
    proto, err := p.getProto(topic)
    if err != nil {
        return fmt.Errorf("msg topic out of range: %v", topic)
    }

    select {
    case proto.in <- msg:
        return nil
    case <-p.closed:
        return io.EOF
    }

    return nil
}

// getProto finds the protocol responsible for handling
// the given message topic.
func (p *Peer) getProto(topic string) (*protoRW, error) {
    for _, proto := range p.running {
        for _, t := range proto.Topics {
            if strings.HasSuffix(topic, t) {
                return proto, nil
            }
        }
    }

    return nil, newPeerError(errInvalidMsgTopic, "%s", topic)
}

// readLoopCount computes the count of protocols' topics.
// One read loop for every topic.
func (p *Peer) readLoopCount() int {
    var count int

    for _, proto := range p.running {
        count += len(proto.Topics)
    }

    return count
}

func (p *Peer) startProtocols(writeStart <-chan struct{}, writeErr chan<- error) {
    p.wg.Add(len(p.running))
    for _, proto := range p.running {
        proto := proto
        proto.closed = p.closed
        proto.wstart = writeStart
        proto.werr = writeErr
        var rw MsgReadWriter = proto
        p.log.Trace(fmt.Sprintf("Starting protocol %s/%d", proto.Name, proto.Version))

        go func() {
            err := proto.Run(p, rw)
            if err == nil {
                p.log.Trace(fmt.Sprintf("Protocol %s/%d returned", proto.Name, proto.Version))
                err = errProtocolReturned
            }  else if err != io.EOF {
                p.log.Trace(fmt.Sprintf("Protocol %s/%d failed", proto.Name, proto.Version), "err", err)
            }

            p.protoErr <- err
            p.wg.Done()
        }()
    }
}

// matchProtocols creates structures for matching named subprotocols.
func matchProtocols(protocols []Protocol, pubsub PublishSubscriber, homeID string) map[string]*protoRW {
    result := make(map[string]*protoRW)

    for _, proto := range protocols {
        result[proto.Name] = &protoRW{Protocol: proto, homeID: homeID, in: make(chan Msg), writer: pubsub}
    }

    return result
}

type protoRW struct {
    Protocol
    homeID string
    in     chan Msg        // receives read messages
    closed <-chan struct{} // receives when peer is shutting down
    wstart <-chan struct{} // receives when write may start
    werr   chan<- error    // for write results
    writer PublishSubscriber
}

func (rw *protoRW) WriteMsg(msg Msg) (err error) {
    select {
    case <-rw.wstart:
        topic := constructTopic(rw.homeID, msg.Topic)
        data, err := ioutil.ReadAll(msg.Payload)
        if err != nil {
            rw.werr <- err
        } else {
            err = rw.writer.Pub(topic, data)
            // Report write status back to Peer.run. It will initiate
            // shutdown if the error is non-nil and unblock the next write
            // otherwise. The calling protocol code should exit for errors
            // as well but we don't want to rely on that.
            rw.werr <- err
        }

    case <-rw.closed:
        err = ErrShuttingDown
    }

    return err
}

func (rw *protoRW) ReadMsg() (Msg, error) {
    select {
    case msg := <-rw.in:
        return msg, nil

    case <-rw.closed:
        return Msg{}, io.EOF

    case <-time.After(readTimeoutThreshold):
        return Msg{}, ErrReadTimeout
    }
}

func constructTopic(id string, topic string) string {
    return id + topic
}
