package store

import (
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
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
	*leveldb.DB
}

func NewEvmAuxStore(db *leveldb.DB) *EvmAuxStore {
	return &EvmAuxStore{DB: db}
}

func (s EvmAuxStore) GetBloomFilter(height uint64) []byte {
	filter, err := s.Get(bloomFilterKey(height), nil)
	if err != nil {
		return nil
	}
	return filter
}

func (s *EvmAuxStore) GetTxHashList(height uint64) ([][]byte, error) {
	protHashList, err := s.Get(evmTxHashKey(height), nil)
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
