package store

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	LevelDBFilename = "blockIndex_db"
	BisMemory       = "Memory"
	BisLevelDB      = "LevelDB"
	BisLegacy       = "Legacy"
)

type BlockIndexStore interface {
	GetHeight(hash []byte) (uint64, error)
	PutHashHeight(hash []byte, height uint64) error
	ClearData()
	Close() error
}

type BlockIndexStoreConfig struct {
	Method string
}

func NewBlockIndexStore(cfg *BlockIndexStoreConfig) (BlockIndexStore, error) {
	switch cfg.Method {
	case BisMemory:
		return NewMemoryBlockIndexStore(), nil
	case BisLevelDB:
		return NewLevelDBBlockIndexStore()
	case BisLegacy:
		return nil, nil
	default:
		return nil, fmt.Errorf("unrecognised block index store method %s", cfg.Method)
	}
}

func DefaultBlockIndesStoreConfig() *BlockIndexStoreConfig {
	return &BlockIndexStoreConfig{
		Method: BisLegacy,
	}
}

type MemoryBlockIndexStore struct {
	hashHeight map[string]uint64
}

func NewMemoryBlockIndexStore() BlockIndexStore {
	return MemoryBlockIndexStore{
		hashHeight: map[string]uint64{},
	}
}

func (ms MemoryBlockIndexStore) GetHeight(hash []byte) (uint64, error) {
	height, ok := ms.hashHeight[string(hash)]
	if !ok {
		return 0, errors.New("block hash not found")
	}
	return height, nil
}

func (ms MemoryBlockIndexStore) PutHashHeight(hash []byte, height uint64) error {
	ms.hashHeight[string(hash)] = height
	return nil
}

func (ms MemoryBlockIndexStore) Close() error {
	return nil
}

func (ms MemoryBlockIndexStore) ClearData() {
	ms.hashHeight = map[string]uint64{}
}

type LevelDBBlockIndexStore struct {
	db *leveldb.DB
}

func NewLevelDBBlockIndexStore() (BlockIndexStore, error) {
	db, err := leveldb.OpenFile(LevelDBFilename, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s", LevelDBFilename)
	}
	return LevelDBBlockIndexStore{
		db: db,
	}, nil
}

func (ls LevelDBBlockIndexStore) GetHeight(hash []byte) (uint64, error) {
	height, err := ls.db.Get(hash, nil)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(height), nil
}

func (ls LevelDBBlockIndexStore) PutHashHeight(hash []byte, height uint64) error {
	hightBuffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(hightBuffer, height)
	return ls.db.Put(hash, hightBuffer, nil)
}

func (ls LevelDBBlockIndexStore) Close() error {
	return ls.db.Close()
}

func (ls LevelDBBlockIndexStore) ClearData() {
	_ = os.RemoveAll(LevelDBFilename)
}
