package store

import (
	"encoding/binary"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/db"
)

const (
	LevelDBFilename = "blockIndex"
)

type BlockIndexStore interface {
	GetBlockHeightByHash(hash []byte) (uint64, error)
	SetBlockHashAtHeight(height uint64, hash []byte)
	Close()
}

type BlockIndexStoreConfig struct {
	Enabled       bool
	DBBackend     string
	DBName        string
	CacheSizeMegs int
}

func DefaultBlockIndexStoreConfig() *BlockIndexStoreConfig {
	return &BlockIndexStoreConfig{
		Enabled:       false,
		DBBackend:     db.GoLevelDBBackend,
		DBName:        "",
		CacheSizeMegs: 256,
	}
}

func NewBlockIndexStore(enabled bool, dbBackend, name, directory string, cacheSizeMegs int, collectMetrics bool) (BlockIndexStore, error) {
	if !enabled {
		return nil, nil
	}
	dbWrapper, err := db.LoadDB(dbBackend, name, directory, cacheSizeMegs, collectMetrics)
	if err != nil {
		return nil, err
	}
	return &BlockIndexStoreDB{dbWrapper, filepath.Join(directory, name+".db")}, nil
}

type BlockIndexStoreDB struct {
	db   db.DBWrapper
	path string
}

func (bis *BlockIndexStoreDB) GetBlockHeightByHash(hash []byte) (uint64, error) {
	height := bis.db.Get(hash)
	if height == nil {
		return 0, errors.New("block hash not found")
	}
	return binary.BigEndian.Uint64(height), nil
}

func (bis *BlockIndexStoreDB) SetBlockHashAtHeight(height uint64, hash []byte) {
	heightBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBuffer, height)
	bis.db.Set(hash, heightBuffer)
}

func (bis *BlockIndexStoreDB) Close() {
	bis.db.Close()
}
