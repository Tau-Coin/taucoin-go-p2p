package p2p

import (

)

// Protocol represents a P2P subprotocol implementation.
type Protocol struct {
    // Name should contain the official protocol name,
    // often a three-letter word.
    Name string

    // Version should contain the version number of the protocol.
    Version uint

    // Topics should contain the topics
    // which should be handled by the protocol.
    Topics []string

    // Run is called in a new goroutine when the protocol has been
    // negotiated with a peer. It should read and write messages from
    // rw. The Payload for each message must be fully consumed.
    //
    // The peer connection is closed when Start returns. It should return
    // any protocol-level error (such as an I/O error) that is
    // encountered.
    Run func(peer *Peer, rw MsgReadWriter) error
}
