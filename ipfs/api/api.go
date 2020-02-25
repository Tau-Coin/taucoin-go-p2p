package api

import (
    "errors"
    "strings"

    "github.com/ethereum/go-ethereum/log"
    ipfsapi "github.com/ipfs/go-ipfs-http-client"

    ma "github.com/multiformats/go-multiaddr"
)

var (
    ErrInvalidCfg = errors.New("invalid api config")
)

type api struct {

    // ipfs daemon api url
    url      string

    // the value of environment variable 'IPFS_PATH'
    ipfspath string

    // ipfs api http client
    api      *ipfsapi.HttpApi
}

var singleton *api

func init() {
    var err error

    singleton, err = newAPI(nil)
    if err != nil {
        log.Root().Crit("new ipfs api crashed")
    }
}

func newAPI(cfg *Config) (*api, error) {

    if cfg == nil {
        cfg = &DefaultConfig
    }

    var (
        httpapi *ipfsapi.HttpApi
        err     error
    )

    if cfg.Url != "" {
        addr, err := ma.NewMultiaddr(strings.TrimSpace(cfg.Url))
        if err != nil {
            return nil, err
        }

        httpapi, err = ipfsapi.NewApi(addr)
        if err != nil {
            return nil, err
        }
    }

    if cfg.Ipfspath != "" {
        httpapi, err = ipfsapi.NewPathApi(cfg.Ipfspath)
        if err != nil {
            return nil, err
        }
    }

    return &api{
            url:      cfg.Url,
            ipfspath: cfg.Ipfspath,
            api:      httpapi,
    }, nil
}

func API() *ipfsapi.HttpApi {
    return singleton.api
}
