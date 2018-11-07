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

func (sr *StateDBReceipts) CommitBlock(state loomchain.State, receipts []*types.EvmTxReceipt, height uint64) error {
	if len(receipts) == 0 {
		return nil
	}

	var txHashArray [][]byte
	var events []*types.EventData
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

func (lr *StateDBReceipts) UpdateReceipt(state loomchain.State, receipt types.EvmTxReceipt) error {
	receiptState := store.PrefixKVStore(loomchain.ReceiptPrefix, state)
	if !receiptState.Has(receipt.TxHash) {
		return errors.Errorf( "cannot find receipt with hash %v", receipt.TxHash)
	}
	protoReceipt, err := proto.Marshal(&receipt)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal receipt with hash %v", receipt.TxHash)
	}
	receiptState.Set(receipt.TxHash, protoReceipt)
	return nil
}