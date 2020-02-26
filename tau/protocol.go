package tau

const (
    // TAU protocol name
    protocolName    = "tau"

    // TAU protocol current version
    protocolVersion = 1
)

// ProtocolVersions are the supported versions of the tau protocol (first is primary).
var ProtocolVersions = []uint{protocolVersion}

// protocolLengths are the number of implemented message corresponding to different protocol versions.
var protocolLengths = map[uint]uint64{protocolVersion: 1}

const (
    // TAU protocol topics
    // which is the topic of libp2p pubsub call.

    // block list
    blockList = "BLKLIST"

    txList    = "TXLIST"
)

var (
    protocolTopics = []string{
            blockList,
            txList,
    }
)
