package ipfsdb

import (
	"bytes"
    "context"

    "github.com/Tau-Coin/taucoin-go-p2p/taudb/utils"

    caopts "github.com/ipfs/interface-go-ipfs-core/options"
	mh     "github.com/multiformats/go-multihash"
    ipfs   "github.com/Tau-Coin/taucoin-go-p2p/ipfs/api"
)

type IPFSdb struct {
    ctx  context.Context
}

func NewIPFSdb(ctx context.Context) *IPFSdb {
    return &IPFSdb{
		ctx:  ctx,
    }
}

func (db *IPFSdb) Put(key, value []byte) error {
	// value -> io.Reader
	reader := bytes.NewReader(value)

	opt := func(bs *caopts.BlockPutSettings) error {
		bs.Codec = "0xa0"
		bs.MhType = mh.KECCAK_256
		bs.MhLength = -1
		bs.Pin = true
		return nil
	}

	_, err:= ipfs.API().Block().Put(db.ctx, reader, opt)

	return err
}

func (db *IPFSdb) Get(key []byte) ([]byte, error) {
	// key -> path
	path, err:= utils.Keccak256ToPath(0xa0, key)
	if err != nil {
		return nil, err
	}

	reader, err:= ipfs.API().Block().Get(db.ctx, path)
	if err != nil{
		return nil, err
	}

    var data []byte
	_, errRead := reader.Read(data)

	return data, errRead
}

func (db *IPFSdb) Delete(key []byte) error {
	// key -> path
	path, err:= utils.Keccak256ToPath(0xa0, key)
	if err != nil {
		return err
	}

    return ipfs.API().Block().Rm(db.ctx, path)
}

func (db *IPFSdb) Has(key []byte) (bool, error) {
	// key -> path
	path, err:= utils.Keccak256ToPath(0xa0, key)
	if err != nil {
		return false, err
	}

	blockStat, err:= ipfs.API().Block().Stat(db.ctx, path)

	return blockStat.Size()> 0, err
}

// TBD
func (db *IPFSdb) Write(batch *Batch) error {
	if batch == nil || batch.Len() == 0 {
        return nil
    }
	for i:= 0; i< batch.internalLen; i++{

		keyStart := batch.index[i].keyPos
		keyEnd := keyStart+ batch.index[i].keyLen

		valueStart := batch.index[i].valuePos
		valueEnd := valueStart+ batch.index[i].valueLen

		keyTmp := batch.data[keyStart : keyEnd]
		valueTmp := batch.data[valueStart : valueEnd]

		err:= db.Put(keyTmp, valueTmp)
		if err != nil {
			return err
		}
	}
	return nil
}
