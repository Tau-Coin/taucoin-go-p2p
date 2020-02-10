package tnode

import (
    "sync"
)

const (
    DefaultSize = 100
)

// DB is not goroutine safe. Because this DB is used in discover routine.
type DB struct {

    mu    sync.Mutex

    // ipfs local node
    home  *Node

    // use map to store ipfs daemon p2p-circuit peers
    nodes map[ID]*Node

    // map capability
    max   uint16
}

func NewDB(max uint16) *DB {
    if max == 0 {
        max = DefaultSize
    }

    return &DB{
        max:   max,
        nodes: make(map[ID]*Node, max),
    }
}

func (db *DB) Close() {
    // nothing to do
}

func (db *DB) Home() *Node {
    db.mu.Lock()
    defer db.mu.Unlock()

    return db.home
}

func (db *DB) SetHome(n *Node) {
    db.mu.Lock()
    defer db.mu.Unlock()

    db.home = n
}

func (db *DB) Get(id ID) *Node {
    db.mu.Lock()
    defer db.mu.Unlock()

    return db.nodes[id]
}

func (db *DB) Put(n *Node) {
    db.mu.Lock()
    defer db.mu.Unlock()

    db.nodes[n.ID()] = n;
}

func (db *DB) Size() int {
    db.mu.Lock()
    defer db.mu.Unlock()

    return len(db.nodes)
}

func (db *DB) Choose(news map[ID]*Node) (added, removed []*Node) {
    db.mu.Lock()
    defer db.mu.Unlock()

    for k, v := range db.nodes {
        if news[k] == nil {
            removed = append(removed, v)
        }
    }

    for k, v := range news {
        if db.nodes[k] == nil {
            added = append(added, v)
        }
    }

    return
}

func (db *DB) Replace(news map[ID]*Node) {
    db.mu.Lock()
    defer db.mu.Unlock()

    db.nodes = news
}
