package evmaux

import (
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	legacyEvmAuxDBName = "receipts_db"

	bloomPrefix     = []byte("bf")
	txHashPrefix    = []byte("th")
	txRefPrefix     = []byte("txr")
	dupTxHashPrefix = []byte("dtx")

	// keys to receipts linked list
	headKey          = []byte("leveldb:head")
	tailKey          = []byte("leveldb:tail")
	currentDbSizeKey = []byte("leveldb:size")
)

func dupTxHashKey(txHash []byte) []byte {
	return util.PrefixKey(dupTxHashPrefix, txHash)
}

func bloomFilterKey(height uint64) []byte {
	return util.PrefixKey(bloomPrefix, blockHeightToBytes(height))
}

func evmTxHashKey(height uint64) []byte {
	return util.PrefixKey(txHashPrefix, blockHeightToBytes(height))
}

func blockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}

func renameReceiptsDB(path, newName string) error {
	if _, err := os.Stat(filepath.Join(path, legacyEvmAuxDBName)); !os.IsNotExist(err) {
		err := os.Rename(filepath.Join(path, legacyEvmAuxDBName), filepath.Join(path, newName+".db"))
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadStore(dbName, rootPath string, maxReceipts uint64) (*EvmAuxStore, error) {
	if maxReceipts == 0 {
		return NewEvmAuxStore(dbm.NewMemDB(), maxReceipts), nil
	}

	// if receipts_db exits, rename it to (default: evmaux.db)
	if err := renameReceiptsDB(rootPath, dbName); err != nil {
		return nil, err
	}
	evmAuxDB, err := dbm.NewGoLevelDB(dbName, rootPath)
	if err != nil {
		return nil, err
	}
	evmAuxStore := NewEvmAuxStore(evmAuxDB, maxReceipts)

	// load duplicate tx hashes from db
	dupEVMTxHashes := make(map[string]bool)
	iter := evmAuxDB.Iterator(
		dupTxHashPrefix, util.PrefixRangeEnd(dupTxHashPrefix),
	)
	defer iter.Close()
	for iter.Valid() {
		dupTxHash, err := util.UnprefixKey(iter.Key(), dupTxHashPrefix)
		if err != nil {
			return nil, err
		}
		dupEVMTxHashes[string(dupTxHash)] = true
		iter.Next()
	}
	evmAuxStore.SetDupEVMTxHashes(dupEVMTxHashes)

	return evmAuxStore, nil
}

// ChildTxRef links a Tendermint tx hash to an EVM tx hash.
type ChildTxRef struct {
	ParentTxHash []byte
	ChildTxHash  []byte
}

type EvmAuxStore struct {
	db             dbm.DB
	maxReceipts    uint64
	dupEVMTxHashes map[string]bool
}

func NewEvmAuxStore(db dbm.DB, maxReceipts uint64) *EvmAuxStore {
	return &EvmAuxStore{
		db:             db,
		maxReceipts:    maxReceipts,
		dupEVMTxHashes: make(map[string]bool),
	}
}

func (s *EvmAuxStore) SetDupEVMTxHashes(dupEVMTxHashes map[string]bool) {
	s.dupEVMTxHashes = dupEVMTxHashes
}

func (s *EvmAuxStore) GetDupEVMTxHashes() map[string]bool {
	return s.dupEVMTxHashes
}

func (s *EvmAuxStore) GetBloomFilter(height uint64) []byte {
	filter := s.db.Get(bloomFilterKey(height))
	if len(filter) == 0 {
		return nil
	}
	return filter
}

func (s *EvmAuxStore) GetTxHashList(height uint64) ([][]byte, error) {
	protHashList := s.db.Get(evmTxHashKey(height))
	if len(protHashList) == 0 {
		return [][]byte{}, nil
	}
	txHashList := types.EthTxHashList{}
	err := proto.Unmarshal(protHashList, &txHashList)
	return txHashList.EthTxHash, err
}

// SaveChildTxRefs persists references between Tendermint & EVM tx hashes to the underlying DB.
func (s *EvmAuxStore) SaveChildTxRefs(refs []ChildTxRef) error {
	if len(refs) == 0 {
		return nil
	}
	batch := s.db.NewBatch()
	defer batch.Close()
	for _, ref := range refs {
		batch.Set(util.PrefixKey(txRefPrefix, ref.ParentTxHash), ref.ChildTxHash)
	}
	batch.Write()
	return nil
}

// GetChildTxHash looks up the EVM tx hash that corresponds to the given Tendermint tx hash.
func (s *EvmAuxStore) GetChildTxHash(parentTxHash []byte) []byte {
	return s.db.Get(util.PrefixKey(txRefPrefix, parentTxHash))
}

// IsDupEVMTxHash checks if the tx hash is in the duplicate tx hash list
func (s *EvmAuxStore) IsDupEVMTxHash(txHash []byte) bool {
	_, ok := s.dupEVMTxHashes[string(txHash)]
	return ok
}

func (s *EvmAuxStore) DB() dbm.DB {
	return s.db
}
