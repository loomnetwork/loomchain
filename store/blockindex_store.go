package store

import (
	"encoding/binary"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/db"
)

const (
	LevelDBFilename = "blockIndex"
	BisLegacy       = "legacy"
)

type BlockIndexStore interface {
	GetBlockHeightByHash(hash []byte) (uint64, error)
	SetBlockHashAtHeight(hash []byte, height uint64)
	Close()
}

type BlockIndexStoreConfig struct {
	Method        string
	Name          string
	CacheSizeMegs int
}

func DefaultBlockIndesStoreConfig() *BlockIndexStoreConfig {
	return &BlockIndexStoreConfig{
		Method:        BisLegacy,
		Name:          "",
		CacheSizeMegs: 0,
	}
}

func NewBlockIndexStore(dbBackend, name, directory string, cacheSizeMegs int, collectMetrics bool) (BlockIndexStore, error) {
	if dbBackend == BisLegacy {
		return nil, nil
	}
	dbWrapper, err := db.LoadDB(dbBackend, name, directory, cacheSizeMegs, collectMetrics)
	if err != nil {
		return nil, err
	}
	return BlockIndexStoreDB{dbWrapper, filepath.Join(directory, name+".db")}, nil
}

type BlockIndexStoreDB struct {
	db   db.DBWrapper
	path string
}

func (bis BlockIndexStoreDB) GetBlockHeightByHash(hash []byte) (uint64, error) {
	height := bis.db.Get(hash)
	if height == nil {
		return 0, errors.New("block hash not found")
	}
	return binary.BigEndian.Uint64(height), nil
}

func (bis BlockIndexStoreDB) SetBlockHashAtHeight(hash []byte, height uint64) {
	hightBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(hightBuffer, height)
	bis.db.Set(hash, hightBuffer)
}

func (bis BlockIndexStoreDB) Close() {
	bis.db.Close()
}
