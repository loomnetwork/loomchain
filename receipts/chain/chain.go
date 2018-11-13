package chain

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

func (sr *StateDBReceipts) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(loomchain.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

type StateDBReceipts struct {
}

func (sr *StateDBReceipts) CommitBlock(state loomchain.State, receipts []*types.EvmTxReceipt, height uint64, blockHash []byte) error {
	if len(receipts) == 0 {
		return nil
	}

	var txHashArray [][]byte
	var events []*types.EventData
	numEvmTxs := int32(0)
	for _, txReceipt := range receipts {
		if txReceipt == nil || len(txReceipt.TxHash) == 0 {
			continue
		}

		txReceipt.BlockHash = blockHash
		if txReceipt.Status == loomchain.StatusTxSuccess {
			txReceipt.TransactionIndex = numEvmTxs
			numEvmTxs++
		}

		postTxReceipt, err := proto.Marshal(txReceipt)
		if err != nil {
			log.Error(fmt.Sprintf("commit block reipts: marshal tx receipt: %s", err.Error()))
			continue
		}
		if txReceipt.Status == loomchain.StatusTxSuccess {
			txHashArray = append(txHashArray, txReceipt.TxHash)
		}
		events = append(events, txReceipt.Logs...)
		receiptState := store.PrefixKVStore(loomchain.ReceiptPrefix, state)
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
