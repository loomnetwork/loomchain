package evmaux

import (
	"encoding/binary"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

var (
	EvmAuxDBName = "receipts_db"

	BloomPrefix  = []byte("bf")
	TxHashPrefix = []byte("th")
	txRefPrefix  = []byte("txr")
)

func bloomFilterKey(height uint64) []byte {
	return util.PrefixKey(BloomPrefix, blockHeightToBytes(height))
}

func evmTxHashKey(height uint64) []byte {
	return util.PrefixKey(TxHashPrefix, blockHeightToBytes(height))
}

func blockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}

func LoadStore() (*EvmAuxStore, error) {
	evmAuxDB, err := goleveldb.OpenFile(EvmAuxDBName, nil)
	if err != nil {
		return nil, err
	}
	return NewEvmAuxStore(evmAuxDB), nil
}

// ChildTxRef links a Tendermint tx hash to an EVM tx hash.
type ChildTxRef struct {
	ParentTxHash []byte
	ChildTxHash  []byte
}

type EvmAuxStore struct {
	db *leveldb.DB
}

func NewEvmAuxStore(db *leveldb.DB) *EvmAuxStore {
	return &EvmAuxStore{db: db}
}

func (s *EvmAuxStore) Close() error {
	return s.db.Close()
}

func (s *EvmAuxStore) GetBloomFilter(height uint64) []byte {
	filter, err := s.db.Get(bloomFilterKey(height), nil)
	if err != nil && err != leveldb.ErrNotFound {
		panic(err)
	}
	if err == leveldb.ErrNotFound {
		return nil
	}
	return filter
}

func (s *EvmAuxStore) GetTxHashList(height uint64) ([][]byte, error) {
	protHashList, err := s.db.Get(evmTxHashKey(height), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if err == leveldb.ErrNotFound {
		return [][]byte{}, nil
	}
	txHashList := types.EthTxHashList{}
	err = proto.Unmarshal(protHashList, &txHashList)
	return txHashList.EthTxHash, err
}

func (s *EvmAuxStore) SetBloomFilter(tran *leveldb.Transaction, filter []byte, height uint64) error {
	return tran.Put(bloomFilterKey(height), filter, nil)
}

func (s *EvmAuxStore) SetTxHashList(tran *leveldb.Transaction, txHashList [][]byte, height uint64) error {
	postTxHashList, err := proto.Marshal(&types.EthTxHashList{EthTxHash: txHashList})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	tran.Put(evmTxHashKey(height), postTxHashList, nil)
	return nil
}

// SaveChildTxRefs persists references between Tendermint & EVM tx hashes to the underlying DB.
func (s *EvmAuxStore) SaveChildTxRefs(refs []ChildTxRef) error {
	if len(refs) == 0 {
		return nil
	}

	tran, err := s.db.OpenTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to open tx in EvmAuxStore")
	}
	defer tran.Discard()

	for _, ref := range refs {
		tran.Put(util.PrefixKey(txRefPrefix, ref.ParentTxHash), ref.ChildTxHash, nil)
	}

	if err := tran.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit tx in EvmAuxStore")
	}

	return nil
}

// GetChildTxHash looks up the EVM tx hash that corresponds to the given Tendermint tx hash.
func (s *EvmAuxStore) GetChildTxHash(parentTxHash []byte) ([]byte, error) {
	return s.db.Get(util.PrefixKey(txRefPrefix, parentTxHash), nil)
}

func (s *EvmAuxStore) DB() *leveldb.DB {
	return s.db
}
func (s *EvmAuxStore) ClearData() {
	os.RemoveAll(EvmAuxDBName)
}
