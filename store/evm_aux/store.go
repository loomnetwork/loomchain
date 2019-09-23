package evmaux

import (
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	legacyEvmAuxDBName = "receipts_db"

	bloomPrefix  = []byte("bf")
	txHashPrefix = []byte("th")
	txRefPrefix  = []byte("txr")

	// keys to receipts linked list
	headKey          = []byte("leveldb:head")
	tailKey          = []byte("leveldb:tail")
	currentDbSizeKey = []byte("leveldb:size")
)

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
	return NewEvmAuxStore(evmAuxDB, maxReceipts), nil
}

// ChildTxRef links a Tendermint tx hash to an EVM tx hash.
type ChildTxRef struct {
	ParentTxHash []byte
	ChildTxHash  []byte
}

type EvmAuxStore struct {
	db          dbm.DB
	batch       dbm.Batch
	maxReceipts uint64
}

func NewEvmAuxStore(db dbm.DB, maxReceipts uint64) *EvmAuxStore {
	return &EvmAuxStore{
		db:          db,
		batch:       db.NewBatch(),
		maxReceipts: maxReceipts,
	}
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

func (s *EvmAuxStore) setBloomFilter(filter []byte, height uint64) {
	s.batch.Set(bloomFilterKey(height), filter)
}

func (s *EvmAuxStore) setTxHashList(txHashList [][]byte, height uint64) error {
	postTxHashList, err := proto.Marshal(&types.EthTxHashList{EthTxHash: txHashList})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	s.batch.Set(evmTxHashKey(height), postTxHashList)
	return nil
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

func (s *EvmAuxStore) DB() dbm.DB {
	return s.db
}

func (s *EvmAuxStore) Commit() {
	s.batch.WriteSync()
	s.Rollback()
}

func (s *EvmAuxStore) Rollback() {
	s.batch = s.db.NewBatch()
}
