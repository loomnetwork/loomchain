package common

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	loom_types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

func GetTxHashList(state loomchain.ReadOnlyState, height uint64) ([][]byte, error) {
	receiptState := store.PrefixKVReader(loomchain.TxHashPrefix, state)
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

	postTxHashList, err := proto.Marshal(&types.EthTxHashList{txHashList})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	txHashState := store.PrefixKVStore(loomchain.TxHashPrefix, state)
	txHashState.Set(BlockHeightToBytes(height), postTxHashList)
	return nil
}

func GetBloomFilter(state loomchain.ReadOnlyState, height uint64) []byte {
	bloomState := store.PrefixKVReader(loomchain.BloomPrefix, state)
	return bloomState.Get(BlockHeightToBytes(height))
}

func SetBloomFilter(state loomchain.State, filter []byte, height uint64) {
	bloomState := store.PrefixKVWriter(loomchain.BloomPrefix, state)
	bloomState.Set(BlockHeightToBytes(height), filter)
}

func WriteReceipt(
	block loom_types.BlockHeader,
	caller, addr loom.Address,
	events []*types.EventData,
	status int32,
	eventHadler loomchain.EventHandler,
) (types.EvmTxReceipt, error) {
	txReceipt := types.EvmTxReceipt{
		TransactionIndex:  block.NumTxs,
		BlockHash:         block.GetLastBlockID().Hash,
		BlockNumber:       block.Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         bloom.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}

	preTxReceipt, err := proto.Marshal(&txReceipt)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling reciept")
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)

	txReceipt.TxHash = txHash
	blockHeight := uint64(txReceipt.BlockNumber)
	for _, event := range events {
		event.TxHash = txHash
		if eventHadler != nil {
			_ = eventHadler.Post(blockHeight, event)
		}
		pEvent := types.EventData(*event)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}

	return txReceipt, nil
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

func ConvertEventData(events []*loomchain.EventData) []*types.EventData {
	var typesEvents []*types.EventData
	for _, event := range events {
		typeEvent := types.EventData(*event)
		typesEvents = append(typesEvents, &typeEvent)
	}
	return typesEvents
}
