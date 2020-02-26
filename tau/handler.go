package tau

import (
    "github.com/Tau-Coin/taucoin-go-p2p/p2p"
)

const (
    maxPeers = 32
)

type ProtocolManager struct {
        peers *peerSet
}

// NewProtocolManager returns a new TAU sub protocol manager. The TAU sub protocol manages peers capable
// with the TAU network.
func NewProtocolManager() *ProtocolManager {
    return &ProtocolManager{
            peers: newPeerSet(),
    }
}

func (pm *ProtocolManager) MakeProtocol(version uint) p2p.Protocol {
    _, ok := protocolLengths[version]
    if !ok {
        panic("makeProtocol for unknown version")
    }

    proto := p2p.Protocol{
            Name:    protocolName,
            Version: version,
	    Run:     func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
                         peer := pm.newPeer(int(version), p, rw)
			 return pm.handle(peer)
	             },
    }
    copy(proto.Topics, protocolTopics)

    return proto
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, rw)
}

func (pm *ProtocolManager) removePeer(id string) {
    // Short circuit if the peer was already removed
    peer := pm.peers.Peer(id)
    if peer == nil {
        return
    }
    //log.Debug("Removing TAU peer", "peer", id)

    if err := pm.peers.Unregister(id); err != nil {
        //log.Error("Peer removal failed", "peer", id, "err", err)
    }

    // Hard disconnect at the networking layer
    if peer != nil {
        peer.Peer.Disconnect(p2p.DiscUselessPeer)
    }
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
    if pm.peers.Len() >= maxPeers {
        return p2p.DiscTooManyPeers
    }

    p.Log().Debug("TAU peer connected")

    // Register the peer locally
    if err := pm.peers.Register(p); err != nil {
        p.Log().Error("TAU peer registration failed", "err", err)
        return err
    }
    defer pm.removePeer(p.id)

    // Handle incoming messages until the connection is torn down
    for {
        if err := pm.handleMsg(p); err != nil {
            p.Log().Debug("TAU message handling failed", "err", err)
            return err
        }
    }
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
    // Read the next message from the remote peer, and ensure it's fully consumed
    msg, err := p.rw.ReadMsg()
    if err != nil {
        return err
    }

    switch {
    case msg.Topic == blockList:

    case msg.Topic == txList:

    default:
    }

    return nil
}
