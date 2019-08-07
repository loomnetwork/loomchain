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
	mutex           *sync.RWMutex
	receiptsCache   []*types.EvmTxReceipt
	tmTxHashIndex   []common.HashPair
	txHashList      [][]byte
	currentReceipt  *types.EvmTxReceipt
}

func NewReceiptHandler(
	eventHandler loomchain.EventHandler,
	maxReceipts uint64, evmAuxStore *evmaux.EvmAuxStore,
) *ReceiptHandler {
	return &ReceiptHandler{
		eventHandler:    eventHandler,
		receiptsCache:   []*types.EvmTxReceipt{},
		txHashList:      [][]byte{},
		tmTxHashIndex:   []common.HashPair{},
		currentReceipt:  nil,
		mutex:           &sync.RWMutex{},
		leveldbReceipts: leveldb.NewLevelDbReceipts(evmAuxStore, maxReceipts),
	}
}

func (r *ReceiptHandler) GetHashFromTmHash(tmHash []byte) ([]byte, error) {
	return r.leveldbReceipts.GetHashFromTmHash(tmHash)
}

func (r *ReceiptHandler) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	// Convert if using tendermint txHash
	loomTxHash, err := r.GetHashFromTmHash(txHash)
	if len(loomTxHash) > 0 && err == nil {
		txHash = loomTxHash
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

func (r *ReceiptHandler) CommitCurrentReceipt(tmHash []byte) {
	if r.currentReceipt != nil {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.receiptsCache = append(r.receiptsCache, r.currentReceipt)
		r.txHashList = append(r.txHashList, r.currentReceipt.TxHash)
		if len(tmHash) > 0 {
			r.tmTxHashIndex = append(r.tmTxHashIndex, common.HashPair{
				tmHash,
				r.currentReceipt.TxHash},
			)
		}
		r.currentReceipt = nil
		r.tmTxHashIndex = []common.HashPair{}
	}
}

func (r *ReceiptHandler) DiscardCurrentReceipt() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.currentReceipt = nil
}

func (r *ReceiptHandler) CommitBlock(state loomchain.State, height int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	err := r.leveldbReceipts.CommitBlock(state, r.receiptsCache, uint64(height), r.tmTxHashIndex)
	r.txHashList = [][]byte{}
	r.receiptsCache = []*types.EvmTxReceipt{}
	return err
}

// TODO: this doesn't need the entire state passed in, just the block header
func (r *ReceiptHandler) CacheReceipt(
	state loomchain.State, caller, addr loom.Address, events []*types.EventData, txErr error,
) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentReceipt != nil {
		r.currentReceipt.Logs = append(r.currentReceipt.Logs, events...)
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
	)
	if err != nil {
		return []byte{}, errors.Wrap(err, "receipt not written, returning empty hash")
	}
	r.currentReceipt = &receipt
	return r.currentReceipt.TxHash, err
}
