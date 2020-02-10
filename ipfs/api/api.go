package api

import (
    "errors"
    "strings"

    ipfsapi"github.com/ipfs/go-ipfs-http-client"

    ma "github.com/multiformats/go-multiaddr"
)

var (
    ErrInvalidCfg = errors.New("invalid api config")
)

type API struct {

    // ipfs daemon api url
    url      string

    // the value of environment variable 'IPFS_PATH'
    ipfspath string

    // ipfs api http client
    api      *ipfsapi.HttpApi
}

func New(cfg *Config) (*API, error) {

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

    return &API{
            url:      cfg.Url,
            ipfspath: cfg.Ipfspath,
            api:      httpapi,
    }, nil
}

func (api *API) HttpAPI() *ipfsapi.HttpApi {
    return api.api
}
