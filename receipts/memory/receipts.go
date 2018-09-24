package memory

import (
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/receipts/common`
	`github.com/pkg/errors`
)

var receipts map[string]types.EvmTxReceipt
var txHashes map[uint64][]byte
var bloomFilters map[uint64][]byte

func init() {
	receipts = make(map[string]types.EvmTxReceipt)
	txHashes = make(map[uint64][]byte)
	bloomFilters = make(map[uint64][]byte)
}

type ReadStateReceipts struct {
}

func (rsr ReadStateReceipts) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	if receipts == nil {
		return types.EvmTxReceipt{}, errors.New("no receipt map")
	}
	return receipts[string(txHash)], nil
}

func (rsr ReadStateReceipts) GetTxHash(height uint64) ([]byte, error) {
	if txHashes == nil {
		return nil, errors.New("no txHash map")
	}
	return txHashes[height], nil
}

func (rsr ReadStateReceipts) GetBloomFilter(height uint64) ([]byte, error) {
	if bloomFilters == nil {
		return nil, errors.New("no bloom filter map")
	}
	return bloomFilters[height], nil
}

type WriteStateReceipts struct {
	State loomchain.State
	EventHandler loomchain.EventHandler
}

func (wsr WriteStateReceipts) SaveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	if receipts == nil || txHashes == nil || bloomFilters == nil {
		return nil, errors.New("no receipt map")
	}
	txReceipt, err := common.WriteReceipt(wsr.State, caller, addr , events , err , wsr.EventHandler)
	if err != nil {
		return []byte{}, err
	}
	
	height := uint64(txReceipt.BlockNumber)
	txHashes[height] = txReceipt.TxHash
	bloomFilters[height] = txReceipt.LogsBloom
	receipts[string(txReceipt.TxHash)] = txReceipt
	return txReceipt.TxHash, err
}
