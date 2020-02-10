package api


const (
    Default_Url = "/ip4/127.0.0.1/tcp/5001"
)

var (
    DefaultConfig = Config{
        Url:      Default_Url,
        Ipfspath: "",
    }
)

type Config struct {
    // ipfs daemon api url
    Url      string

    // the value of environment variable 'IPFS_PATH'
    Ipfspath string
}
