package util

import (
    cid "github.com/ipfs/go-cid"
    "github.com/ipfs/interface-go-ipfs-core/path"
)


func ByteToPath(b []byte) (path.Path, error){
	// byte into cid
	_, cid, err := cid.CidFromBytes(b)
	if err != nil{
       return nil, err
	}

	// cid into path
	res := path.IpfsPath(cid)

	return res.Path, nil
}
