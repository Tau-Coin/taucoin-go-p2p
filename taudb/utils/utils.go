package utils

import (
    cid "github.com/ipfs/go-cid"
    "github.com/ipfs/interface-go-ipfs-core/path"
)


func ByteToPath(b []byte) (path.Path, error){
	// byte into cid
	c, err := cid.Decode(string(b))
	if err != nil{
       return nil, err
	}

	// cid into path
	res := path.IpfsPath(c)

	return res, nil
}
