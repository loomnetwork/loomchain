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

type ReadStateReceipts struct {
	State loomchain.ReadOnlyState
}

func (rsr ReadStateReceipts) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(receipts.ReceiptPrefix, rsr.State)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

func (rsr ReadStateReceipts) GetTxHash(height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.TxHashPrefix, rsr.State)
	txHash := receiptState.Get(common.BlockHeightToBytes(height))
	return txHash, nil
}

func (rsr ReadStateReceipts) GetBloomFilter(height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.BloomPrefix, rsr.State)
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
