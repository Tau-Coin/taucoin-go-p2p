package forum

import (
    "bytes"
    "fmt"
    "time"

    "github.com/Tau-Coin/taucoin-go-p2p/p2p"
)

const (
    protocolName    = "forum"

    protocolVersion = 1

    defHelloInterval = 15 * time.Second
)

var (
    protocolTopics = []string{"HELLO", "TIME", "TIMERSP"}
)

func MakeForum() p2p.Protocol {
    return p2p.Protocol {
        Name: protocolName,
        Version: protocolVersion,
        Topics:  protocolTopics,
        Run:     run,
    }
}

func run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
    peer.Log().Info("Peer bringup")
    go helloLoop(peer, rw)
    return handler(peer, rw)
}

func handler(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
    for {
        msg, err := rw.ReadMsg()

    if err != nil {
        return err
    }

    peer.Log().Info("forum", "Receive message:", msg)

    switch msg.Topic {
    case "TIME":
        resp := p2p.Msg{
                Topic:  "TIMERSP",
                Payload:bytes.NewReader([]byte(fmt.Sprintf("Now:%s", time.Now()))),
        }

        errw := rw.WriteMsg(resp)
        if errw != nil {
                peer.Log().Error("Write error ", errw)
        }
    }
    }
}

func helloLoop(peer *p2p.Peer, rw p2p.MsgReadWriter) {
    ticker := time.NewTicker(defHelloInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            sayhello(peer, rw)
	}
    }
}

func sayhello(peer *p2p.Peer, rw p2p.MsgReadWriter) {
    msg := p2p.Msg{
        Topic:   "HELLO",
        Payload: bytes.NewReader([]byte("hello " + string(peer.ID()))),
    }

    err := rw.WriteMsg(msg)
    if err != nil {
        peer.Log().Error("Write error ", err)
    }
}
