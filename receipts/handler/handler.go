package handler

import (
	"bytes"
	"sync"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/pkg/errors"
)

type ReceiptHandlerVersion int32

const (
	ReceiptHandlerLevelDb = 2 //ctypes.ReceiptStorage_LEVELDB
	DefaultMaxReceipts    = uint64(2000)
)

// ReceiptHandler implements loomchain.ReadReceiptHandler, loomchain.WriteReceiptHandler, and
// loomchain.ReceiptHandlerStore interfaces.
type ReceiptHandler struct {
	eventHandler    loomchain.EventHandler
	leveldbReceipts *leveldb.LevelDbReceipts
	evmAuxStore     *evmaux.EvmAuxStore

	mutex          *sync.RWMutex
	receiptsCache  []*types.EvmTxReceipt
	txHashList     [][]byte
	currentReceipt *types.EvmTxReceipt
}

func NewReceiptHandler(
	eventHandler loomchain.EventHandler,
	maxReceipts uint64, evmAuxStore *evmaux.EvmAuxStore,
) *ReceiptHandler {
	return &ReceiptHandler{
		eventHandler:    eventHandler,
		receiptsCache:   []*types.EvmTxReceipt{},
		txHashList:      [][]byte{},
		currentReceipt:  nil,
		mutex:           &sync.RWMutex{},
		leveldbReceipts: leveldb.NewLevelDbReceipts(evmAuxStore, maxReceipts),
		evmAuxStore:     evmAuxStore,
	}
}

// GetReceipt looks up an EVM tx receipt by tx hash.
// The tx hash can either be the hash of the Tendermint tx within which the EVM tx was embedded or,
// the hash of the embedded EVM tx itself.
func (r *ReceiptHandler) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	// At first assume the input hash is a Tendermint tx hash and try to resolve it to an EVM tx hash,
	// if that fails it might be an EVM tx hash.
	evmTxHash, err := r.evmAuxStore.GetChildTxHash(txHash)
	if len(evmTxHash) > 0 && err == nil {
		txHash = evmTxHash
	}

	receipt, err := r.leveldbReceipts.GetReceipt(txHash)
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

func (r *ReceiptHandler) Close() error {
	if err := r.leveldbReceipts.Close(); err != nil {
		return errors.Wrap(err, "closing receipt leveldb")
	}
	return nil
}

func (r *ReceiptHandler) ClearData() error {
	r.leveldbReceipts.ClearData()
	return nil
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

	err := r.leveldbReceipts.CommitBlock(r.receiptsCache, uint64(height))
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
			leveldb.CreateEventLogs(r.currentReceipt, state.Block(), events, r.eventHandler)...,
		)
		return r.currentReceipt.TxHash, nil
	}

	var status int32
	if txErr == nil {
		status = common.StatusTxSuccess
	} else {
		status = common.StatusTxFail
	}

	receipt, err := leveldb.WriteReceipt(
		state.Block(), caller, addr, events, status,
		r.eventHandler, int32(len(r.receiptsCache)), int64(auth.Nonce(state, caller)),
		txHash,
	)
	if err != nil {
		return []byte{}, errors.Wrap(err, "receipt not written, returning empty hash")
	}

	r.currentReceipt = &receipt
	return r.currentReceipt.TxHash, err
}
