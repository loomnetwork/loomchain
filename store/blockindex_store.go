package store

import (
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/db"
)

const (
	LevelDBFilename = "block_index"
)

var (
	ErrNotFound = errors.New("block hash not found")
)

type BlockIndexStore interface {
	GetBlockHeightByHash(hash []byte) (uint64, error)
	SetBlockHashAtHeight(height uint64, hash []byte)
	Close()
}

type BlockIndexStoreConfig struct {
	Enabled         bool
	DBBackend       string
	DBName          string
	CacheSizeMegs   int
	WriteBufferMegs int
}

func DefaultBlockIndexStoreConfig() *BlockIndexStoreConfig {
	return &BlockIndexStoreConfig{
		Enabled:         false,
		DBBackend:       db.GoLevelDBBackend,
		DBName:          "block_index",
		CacheSizeMegs:   256,
		WriteBufferMegs: 128,
	}
}

func NewBlockIndexStore(dbBackend, name, directory string, cacheSizeMegs, WriteBufferMegs int, collectMetrics bool) (BlockIndexStore, error) {
	dbWrapper, err := db.LoadDB(dbBackend, name, directory, cacheSizeMegs, WriteBufferMegs, collectMetrics)
	if err != nil {
		return nil, err
	}
	return &BlockIndexStoreDB{dbWrapper}, nil
}

type BlockIndexStoreDB struct {
	db db.DBWrapper
}

func (bis *BlockIndexStoreDB) GetBlockHeightByHash(hash []byte) (uint64, error) {
	height := bis.db.Get(hash)
	if height == nil {
		return 0, ErrNotFound
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
