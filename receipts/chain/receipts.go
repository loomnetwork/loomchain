package chain

import (
	"github.com/gogo/protobuf/proto"
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/receipts/common`
	`github.com/loomnetwork/loomchain/store`
	`github.com/pkg/errors`
)

type ReadStateReceipts struct {
	State loomchain.ReadOnlyState
}

func (rsr ReadStateReceipts) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(loomchain.ReceiptPrefix, rsr.State)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

type WriteStateReceipts struct {
	State loomchain.State
}

func (wsr WriteStateReceipts) Commit(txReceipt types.EvmTxReceipt) error {
	err := common.AppendTxHash(txReceipt.TxHash, wsr.State, uint64(txReceipt.BlockNumber))
	if err != nil {
		return errors.Wrap(err, "appending txHash to state")
	}
	
	postTxReceipt, err := proto.Marshal(&txReceipt)
	if err != nil {
		return errors.Wrap(err, "marshal tx receipt")
	}
	receiptState := store.PrefixKVStore(loomchain.ReceiptPrefix, wsr.State)
	receiptState.Set(txReceipt.TxHash, postTxReceipt)

	return nil
}

func (wsr WriteStateReceipts) ClearData() error {
	return nil
}