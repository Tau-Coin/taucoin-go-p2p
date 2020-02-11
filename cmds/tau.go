package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/ethereum/go-ethereum/log"

    "github.com/Tau-Coin/taucoin-go-p2p/forum"
    "github.com/Tau-Coin/taucoin-go-p2p/p2p"
)

func main() {

    // Create logger
    logger := log.Root()
    logger.SetHandler(log.StdoutHandler)

    cfg := p2p.Config{
        Ctx:             context.Background(),
        MaxPeers:        10,
        EnableMsgEvents: true,
        Logger:          logger,
    }

    // Register protocols
    cfg.Protocols = make([]p2p.Protocol, 0)
    forum := forum.MakeForum()
    cfg.Protocols = append(cfg.Protocols, forum)

    srv := &p2p.Server{Config: cfg}
    srv.Start()

    // listen to the quit signal
    sigCh := make(chan os.Signal)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)
    fmt.Println("receive signal ", <-sigCh)

    srv.Stop()
}
