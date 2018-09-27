package chain

import (
	"github.com/gogo/protobuf/proto"
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/receipts`
	`github.com/loomnetwork/loomchain/receipts/common`
	`github.com/loomnetwork/loomchain/store`
	`github.com/pkg/errors`
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
	State loomchain.State
	EventHandler loomchain.EventHandler
}

func (wsr WriteStateReceipts) SaveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	txReceipt, errWrite := common.WriteReceipt(wsr.State, caller, addr , events , err , wsr.EventHandler)
	if errWrite != nil {
		if err == nil {
			return nil, errors.Wrap(errWrite, "writing receipt")
		} else {
			return nil, errors.Wrapf(err, "error writing reciept %v", errWrite)
		}
	}
	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return nil, errors.Wrap(errMarshal, "marhsal tx receipt")
		} else {
			return nil, errors.Wrapf(err, "marshalling reciept err %v", errMarshal)
		}
	}
	height := common.BlockHeightToBytes(uint64(txReceipt.BlockNumber))
	bloomState := store.PrefixKVStore(receipts.BloomPrefix, wsr.State)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(receipts.TxHashPrefix, wsr.State)
	txHashState.Set(height, txReceipt.TxHash)
	
	receiptState := store.PrefixKVStore(receipts.ReceiptPrefix, wsr.State)
	receiptState.Set(txReceipt.TxHash, postTxReceipt)
	
	return txReceipt.TxHash, err
}

func (wsr WriteStateReceipts) ClearData() error {
	return nil
}