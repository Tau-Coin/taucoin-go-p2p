package discover

import (
    "strings"

    "github.com/ipfs/interface-go-ipfs-core"
)

const (
    P2pCircuit = "p2p-circuit"
)

var _ Filter = &basicFilter{}

type Filter interface {
    Filter([]iface.ConnectionInfo) []iface.ConnectionInfo
}

type basicFilter struct {}

func newFilter() Filter {
    return &basicFilter{}
}

func (f *basicFilter) Filter(conns []iface.ConnectionInfo) []iface.ConnectionInfo {

    var wanted []iface.ConnectionInfo

    for _, c := range(conns) {
        if isP2pCircuit(c) || isPrivate(c) {
            wanted = append(wanted, c)
        }
    }

    return wanted
}

func isP2pCircuit(c iface.ConnectionInfo) bool {
    addrstr := c.Address().String()
    return strings.Contains(addrstr, P2pCircuit)
}

func isPrivate(c iface.ConnectionInfo) bool {
    addrstr := c.Address().String()
    s := strings.Split(addrstr, "/")

    if len(s) >= 2 && isPrivateIP(s[1]) {
        return true
    }

    return false
}

func isPrivateIP(ip string) bool {
    return strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.") || strings.HasPrefix(ip, "192.")
}
