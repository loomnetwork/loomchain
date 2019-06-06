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
	if protHashList == nil {
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

func (s *EvmAuxStore) DB() *leveldb.DB {
	return s.db
}
func (s *EvmAuxStore) ClearData() {
	os.RemoveAll(EvmAuxDBName)
}
