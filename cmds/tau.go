package main

import (
    "context"

    "github.com/Tau-Coin/taucoin-go-p2p/p2p"
)

func main() {
    cfg := p2p.Config{
        Ctx:             context.Background(),
        MaxPeers:        10,
        EnableMsgEvents: true,
    }

    srv := &p2p.Server{Config: cfg}
    srv.Start()

    select {}
}
