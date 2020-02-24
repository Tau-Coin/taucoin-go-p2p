package taudb

import (
    "context"

    "github.com/ipfs/interface-go-ipfs-core"
    "github.com/ipfs/interface-go-ipfs-core/path"

    "github.com/Tau-Coin/taucoin-go-p2p/taudb/utils"
)

type IPFSdb struct {
    ctx  context.Context
    ipfs *api.API
}

func NewIPFSdb(ctx context.Context, ipfs *api.API) *IPFSdb {
    return &IPFSdb{
		ctx:  ctx,
		ipfs: ipfs,
    }
}

func (db *IPFSdb) Put(key, value []byte) error {
	// value -> io.Reader
	reader := bytes.NewReader(value)

	blockstat, err:= db.ipfs.HttpAPI().Block().Put(db.ctx, reader)

	return err
}

func (db *IPFSdb) Get(key []byte) ([]byte, error) {
	// key -> path
	path := utils.ByteToPath(key)

	reader, err:= db.ipfs.HttpAPI().Block().Get(db.ctx, path)
	if err != nil{
		return nil, err
	}

    var data []byte
	_, errRead := reader.Read(data)

	return data, errRead
}

func (db *IPFSdb) Delete(key []byte) error {
	// key -> path
	path := utils.ByteToPath(key)

    return db.ipfs.HttpAPI().Block().Rm(db.ctx, path)
}

func (db *IPFSdb) Has(key []byte) (bool, error) {
	// key -> path
	path := utils.ByteToPath(key)

	blockStat, err:= db.ipfs.HttpAPI().Block().Stat(db.ctx, path)

	return blockStat.Size()> 0, err
}
