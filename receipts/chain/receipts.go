package chain

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

func (wsr WriteStateReceipts) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(receipts.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

func (wsr WriteStateReceipts) GetTxHash(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.TxHashPrefix, state)
	txHash := receiptState.Get(common.BlockHeightToBytes(height))
	return txHash, nil
}

func (wsr WriteStateReceipts) GetBloomFilter(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.BloomPrefix, state)
	boomFilter := receiptState.Get(common.BlockHeightToBytes(height))
	return boomFilter, nil
}

type WriteStateReceipts struct {
	EventHandler loomchain.EventHandler
}

func (wsr WriteStateReceipts) SaveEventsAndHashReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	txReceipt, errWrite := common.WriteReceipt(state, caller, addr, events, err, wsr.EventHandler)
	if errWrite != nil {
		return nil, errors.Wrap(errWrite, "writing receipt")
	}
	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		return nil, errors.Wrap(errMarshal, "marhsal tx receipt")
	}
	height := common.BlockHeightToBytes(uint64(txReceipt.BlockNumber))
	bloomState := store.PrefixKVStore(receipts.BloomPrefix, state)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(receipts.TxHashPrefix, state)
	txHashState.Set(height, txReceipt.TxHash)

	receiptState := store.PrefixKVStore(receipts.ReceiptPrefix, state)
	receiptState.Set(txReceipt.TxHash, postTxReceipt)

	return txReceipt.TxHash, nil
}

func (wsr WriteStateReceipts) ClearData() error {
	return nil
}

func (wsr WriteStateReceipts) Close() {
	//noop
}
