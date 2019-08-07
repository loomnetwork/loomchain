package common

import (
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

const (
	StatusTxSuccess = int32(1)
	StatusTxFail    = int32(0)
)

var (
	ReceiptPrefix = []byte("receipt")
	BloomPrefix   = []byte("bloomFilter")
	TxHashPrefix  = []byte("txHash")

	ErrTxReceiptNotFound = errors.New("Tx receipt not found")
)

func GetTxHashList(state loomchain.ReadOnlyState, height uint64) ([][]byte, error) {
	receiptState := store.PrefixKVReader(TxHashPrefix, state)
	protHashList := receiptState.Get(BlockHeightToBytes(height))
	txHashList := types.EthTxHashList{}
	err := proto.Unmarshal(protHashList, &txHashList)
	return txHashList.EthTxHash, err
}

func AppendTxHashList(state loomchain.State, txHash [][]byte, height uint64) error {
	txHashList, err := GetTxHashList(state, height)
	if err != nil {
		return errors.Wrap(err, "getting tx hash list")
	}
	txHashList = append(txHashList, txHash...)

	postTxHashList, err := proto.Marshal(&types.EthTxHashList{EthTxHash: txHashList})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	txHashState := store.PrefixKVStore(TxHashPrefix, state)
	txHashState.Set(BlockHeightToBytes(height), postTxHashList)
	return nil
}

func GetBloomFilter(state loomchain.ReadOnlyState, height uint64) []byte {
	bloomState := store.PrefixKVReader(BloomPrefix, state)
	return bloomState.Get(BlockHeightToBytes(height))
}

func SetBloomFilter(state loomchain.State, filter []byte, height uint64) {
	bloomState := store.PrefixKVWriter(BloomPrefix, state)
	bloomState.Set(BlockHeightToBytes(height), filter)
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

func ConvertEventData(events []*loomchain.EventData) []*types.EventData {

	typesEvents := make([]*types.EventData, 0, len(events))
	for _, event := range events {
		typeEvent := types.EventData(*event)
		typesEvents = append(typesEvents, &typeEvent)
	}
	return typesEvents
}

type HashPair struct {
	TmTxHash   []byte
	LoomTxHash []byte
}
