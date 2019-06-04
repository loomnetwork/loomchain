package blockindex

import (
	"encoding/binary"

	"github.com/loomnetwork/loomchain/db"
	"github.com/pkg/errors"
)

const (
	// LevelDBFilename is the default name for the block index store DB
	LevelDBFilename = "block_index"
)

var (
	// ErrNotFound is returned by BlockIndexStore.GetBlockHeightByHash() when the index doesn't
	// have an entry for the given hash.
	ErrNotFound = errors.New("block hash not found")
)

func hashKey(hash []byte) []byte {
	return append([]byte("BH:"), hash...)
}

// BlockIndexStore persists block hash -> height index to a DB.
type BlockIndexStore interface {
	// GetBlockHeightByHash looks up the height matching the given block hash.
	GetBlockHeightByHash(hash []byte) (uint64, error)
	// SetBlockHashAtHeight stores the block hash at the given height.
	SetBlockHashAtHeight(height uint64, hash []byte)
	Close()
}

// BlockIndexStoreConfig contains settings for the block index store.
type BlockIndexStoreConfig struct {
	Enabled         bool
	DBBackend       string
	DBName          string
	CacheSizeMegs   int
	WriteBufferMegs int
}

// DefaultBlockIndexStoreConfig returns the default config for the block index store.
func DefaultBlockIndexStoreConfig() *BlockIndexStoreConfig {
	return &BlockIndexStoreConfig{
		Enabled:         false,
		DBBackend:       db.GoLevelDBBackend,
		DBName:          "block_index",
		CacheSizeMegs:   256,
		WriteBufferMegs: 128,
	}
}

// NewBlockIndexStore returns a new instance of the store backed by the given DB.
func NewBlockIndexStore(
	dbBackend, name, directory string,
	cacheSizeMegs, writeBufferMegs int, collectMetrics bool,
) (BlockIndexStore, error) {
	dbWrapper, err := db.LoadDB(dbBackend, name, directory, cacheSizeMegs, writeBufferMegs, collectMetrics)
	if err != nil {
		return nil, err
	}
	return &blockIndexStoreDB{dbWrapper}, nil
}

type blockIndexStoreDB struct {
	db db.DBWrapper
}

func (bis *blockIndexStoreDB) GetBlockHeightByHash(hash []byte) (uint64, error) {
	height := bis.db.Get(hashKey(hash))
	if height == nil {
		return 0, ErrNotFound
	}
	return binary.BigEndian.Uint64(height), nil
}

func (bis *blockIndexStoreDB) SetBlockHashAtHeight(height uint64, hash []byte) {
	heightBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBuffer, height)
	bis.db.Set(hashKey(hash), heightBuffer)
}

func (bis *blockIndexStoreDB) Close() {
	bis.db.Close()
}
