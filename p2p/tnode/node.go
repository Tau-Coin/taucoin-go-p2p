package tnode

import (
    "fmt"

    "github.com/ipfs/interface-go-ipfs-core"
    "github.com/libp2p/go-libp2p-core/protocol"
    ma "github.com/multiformats/go-multiaddr"
)

type ID string

type Node struct {

    // ipfs node identity peer id.
    id ID

    // ipfs node address
    addr ma.Multiaddr

    // protocols
    pids []protocol.ID
}

func New(con iface.ConnectionInfo) *Node {
    n := &Node{
            id:   ID(con.ID()),
            addr: con.Address(),
    }

    pids, _ := con.Streams()
    copy(n.pids[:], pids[:])

    return n
}

func NewNode(id string, addr ma.Multiaddr, pids []protocol.ID) *Node {
    n := &Node{
            id:   ID(id),
            addr: addr,
    }

    if pids != nil {
        copy(n.pids[:], pids[:])
    }

    return n
}

func MakeNodesMap(conns []iface.ConnectionInfo) map[ID]*Node {
    nodes := make(map[ID]*Node)

    for _, c := range conns {
        n := New(c)
        nodes[n.ID()] = n
    }

    return nodes
}

func (n *Node) ID() ID {
    return ID(n.id)
}

func (n *Node) Addr() ma.Multiaddr {
    return n.addr
}

func (n *Node) String() string {
    return fmt.Sprintf("node(id:%s, addr:%s)", n.id, n.addr)
}
