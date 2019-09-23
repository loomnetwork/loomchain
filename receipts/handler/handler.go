package handler

import (
	"bytes"
	"crypto/sha256"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/pkg/errors"
)

type ReceiptHandlerVersion int32

const (
	ReceiptHandlerLevelDb = 2 //ctypes.ReceiptStorage_LEVELDB
)

// ReceiptHandler implements loomchain.ReadReceiptHandler, loomchain.WriteReceiptHandler, and
// loomchain.ReceiptHandlerStore interfaces.
type ReceiptHandler struct {
	eventHandler loomchain.EventHandler
	evmAuxStore  *evmaux.EvmAuxStore

	mutex          *sync.RWMutex
	receiptsCache  []*types.EvmTxReceipt
	txHashList     [][]byte
	currentReceipt *types.EvmTxReceipt
}

func NewReceiptHandler(
	eventHandler loomchain.EventHandler, evmAuxStore *evmaux.EvmAuxStore,
) *ReceiptHandler {
	return &ReceiptHandler{
		eventHandler:   eventHandler,
		receiptsCache:  []*types.EvmTxReceipt{},
		txHashList:     [][]byte{},
		currentReceipt: nil,
		mutex:          &sync.RWMutex{},
		evmAuxStore:    evmAuxStore,
	}
}

// GetReceipt looks up an EVM tx receipt by tx hash.
// The tx hash can either be the hash of the Tendermint tx within which the EVM tx was embedded or,
// the hash of the embedded EVM tx itself.
func (r *ReceiptHandler) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	// At first assume the input hash is a Tendermint tx hash and try to resolve it to an EVM tx hash,
	// if that fails it might be an EVM tx hash.
	evmTxHash := r.evmAuxStore.GetChildTxHash(txHash)
	if len(evmTxHash) > 0 {
		txHash = evmTxHash
	}

	receipt, err := r.evmAuxStore.GetReceipt(txHash)
	if err != nil {
		return receipt, errors.Wrapf(common.ErrTxReceiptNotFound, "GetReceipt: %v", err)
	}
	return receipt, nil
}

func (r *ReceiptHandler) GetPendingReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, receipt := range r.receiptsCache {
		if 0 == bytes.Compare(receipt.TxHash, txHash) {
			return *receipt, nil
		}
	}
	return types.EvmTxReceipt{}, common.ErrPendingReceiptNotFound
}

func (r *ReceiptHandler) GetCurrentReceipt() *types.EvmTxReceipt {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.currentReceipt
}

func (r *ReceiptHandler) GetPendingTxHashList() [][]byte {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	hashListCopy := make([][]byte, len(r.txHashList))
	copy(hashListCopy, r.txHashList)
	return hashListCopy
}

func (r *ReceiptHandler) CommitCurrentReceipt() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.currentReceipt != nil {
		r.receiptsCache = append(r.receiptsCache, r.currentReceipt)
		r.txHashList = append(r.txHashList, r.currentReceipt.TxHash)
		r.currentReceipt = nil
	}
}

func (r *ReceiptHandler) DiscardCurrentReceipt() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.currentReceipt = nil
}

func (r *ReceiptHandler) CommitBlock(height int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	err := r.evmAuxStore.CommitReceipts(r.receiptsCache, uint64(height))
	r.txHashList = [][]byte{}
	r.receiptsCache = []*types.EvmTxReceipt{}
	return err
}

// TODO: this doesn't need the entire state passed in, just the block header
func (r *ReceiptHandler) CacheReceipt(
	state loomchain.State, caller, addr loom.Address, events []*types.EventData, txErr error, txHash []byte,
) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// If there's an existing receipt that means we're trying to store a receipt for an internal
	// contract call, instead of creating a separate receipt for the internal call we merge the logs
	// from the internal call into the existing receipt.
	if r.currentReceipt != nil {
		r.currentReceipt.Logs = append(
			r.currentReceipt.Logs,
			createEventLogs(r.currentReceipt, state.Block(), events, r.eventHandler)...,
		)
		return r.currentReceipt.TxHash, nil
	}

	var status int32
	if txErr == nil {
		status = common.StatusTxSuccess
	} else {
		status = common.StatusTxFail
	}
	receipt, err := writeReceipt(
		state.Block(), caller, addr, events, status, r.eventHandler,
		int32(len(r.receiptsCache)), int64(auth.Nonce(state, caller)), txHash,
	)
	if err != nil {
		return []byte{}, errors.Wrap(err, "receipt not written, returning empty hash")
	}
	r.currentReceipt = &receipt
	return r.currentReceipt.TxHash, err
}

func writeReceipt(
	block loom.BlockHeader,
	caller, addr loom.Address,
	events []*types.EventData,
	status int32,
	eventHandler loomchain.EventHandler,
	evmTxIndex int32,
	nonce int64,
	txHash []byte,
) (types.EvmTxReceipt, error) {
	txReceipt := types.EvmTxReceipt{
		Nonce:             nonce,
		TransactionIndex:  evmTxIndex,
		BlockHash:         block.CurrentHash,
		BlockNumber:       block.Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         bloom.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}

	if len(txHash) == 0 {
		preTxReceipt, err := proto.Marshal(&txReceipt)
		if err != nil {
			return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling receipt")
		}
		h := sha256.New()
		h.Write(preTxReceipt)
		txReceipt.TxHash = h.Sum(nil)
	} else {
		txReceipt.TxHash = txHash
	}

	txReceipt.Logs = append(txReceipt.Logs, createEventLogs(&txReceipt, block, events, eventHandler)...)
	txReceipt.TransactionIndex = block.NumTxs - 1
	return txReceipt, nil
}

func createEventLogs(
	txReceipt *types.EvmTxReceipt,
	block loom.BlockHeader,
	events []*types.EventData,
	eventHandler loomchain.EventHandler,
) []*types.EventData {
	logs := make([]*types.EventData, 0, len(events))
	for _, event := range events {
		event.TxHash = txReceipt.TxHash
		// TODO: Move this out to the caller of CacheReceipt, and decouple EventHandler from ReceiptHandler
		if eventHandler != nil {
			_ = eventHandler.Post(uint64(txReceipt.BlockNumber), event)
		}

		pEvent := types.EventData(*event)
		pEvent.BlockHash = block.CurrentHash
		pEvent.TransactionIndex = uint64(block.NumTxs - 1)
		logs = append(logs, &pEvent)
	}
	return logs
}
