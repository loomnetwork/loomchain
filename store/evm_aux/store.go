package evmaux

import (
	"encoding/binary"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	Db_Filename  = "receipts_db"
	BloomPrefix  = []byte("bloomFilter")
	TxHashPrefix = []byte("txHash")
)

func bloomFilterKey(height uint64) []byte {
	return util.PrefixKey(BloomPrefix, BlockHeightToBytes(height))
}

func evmTxHashKey(height uint64) []byte {
	return util.PrefixKey(TxHashPrefix, BlockHeightToBytes(height))
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

type EvmAuxStore struct {
	db *leveldb.DB
}

func NewEvmAuxStore(db *leveldb.DB) *EvmAuxStore {
	return &EvmAuxStore{db: db}
}

func (s *EvmAuxStore) Get(key []byte) ([]byte, error) {
	return s.db.Get(key, nil)
}

func (s *EvmAuxStore) Set(key, val []byte) error {
	return s.db.Put(key, val, nil)
}

func (s *EvmAuxStore) Has(key []byte) (bool, error) {
	return s.db.Has(key, nil)
}

func (s *EvmAuxStore) Close() error {
	return s.db.Close()
}

func (s *EvmAuxStore) GetBloomFilter(height uint64) []byte {
	filter, err := s.db.Get(bloomFilterKey(height), nil)
	if err != nil {
		return nil
	}
	return filter
}

func (s *EvmAuxStore) GetTxHashList(height uint64) ([][]byte, error) {
	protHashList, err := s.db.Get(evmTxHashKey(height), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	txHashList := types.EthTxHashList{}
	err = proto.Unmarshal(protHashList, &txHashList)
	return txHashList.EthTxHash, err
}

func (s *EvmAuxStore) SetBloomFilter(tran *leveldb.Transaction, filter []byte, height uint64) error {
	return tran.Put(bloomFilterKey(height), filter, nil)
}

func (s *EvmAuxStore) AppendTxHashList(tran *leveldb.Transaction, txHash [][]byte, height uint64) error {
	txHashList, err := s.GetTxHashList(height)
	if err != nil {
		return errors.Wrap(err, "getting tx hash list")
	}
	txHashList = append(txHashList, txHash...)

	postTxHashList, err := proto.Marshal(&types.EthTxHashList{EthTxHash: txHashList})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	tran.Put(evmTxHashKey(height), postTxHashList, nil)
	return nil
}

func (s *EvmAuxStore) DB() *leveldb.DB {
	return s.db
}
func (s *EvmAuxStore) ClearData() {
	os.RemoveAll(Db_Filename)
}
