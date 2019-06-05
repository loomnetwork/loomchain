package chain

import (
	"crypto/sha256"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	loom_types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

func DepreciatedWriteReceipt(
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
		return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling receipt")
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

func (sr *StateDBReceipts) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(common.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	if txReceiptProto == nil || len(txReceiptProto) == 0 {
		return txReceipt, common.ErrTxReceiptNotFound
	}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

type StateDBReceipts struct {
}

func (sr *StateDBReceipts) CommitBlock(state loomchain.State, receipts []*types.EvmTxReceipt, height uint64) error {
	if len(receipts) == 0 {
		return nil
	}

	var txHashArray [][]byte

	events := make([]*types.EventData, 0, len(receipts))
	for _, txReceipt := range receipts {
		if txReceipt == nil || len(txReceipt.TxHash) == 0 {
			continue
		}
		postTxReceipt, err := proto.Marshal(txReceipt)
		if err != nil {
			log.Error(fmt.Sprintf("commit block reipts: marshal tx receipt: %s", err.Error()))
			continue
		}
		txHashArray = append(txHashArray, (*txReceipt).TxHash)
		events = append(events, txReceipt.Logs...)
		receiptState := store.PrefixKVStore(common.ReceiptPrefix, state)
		receiptState.Set(txReceipt.TxHash, postTxReceipt)
	}
	if err := common.AppendTxHashList(state, txHashArray, height); err != nil {
		return errors.Wrap(err, "saving block's tx hash list: %s")
	}
	filter := bloom.GenBloomFilter(events)
	common.SetBloomFilter(state, filter, height)
	return nil
}

func (sr *StateDBReceipts) ClearData() {}
