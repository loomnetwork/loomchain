package handler

import (
	"bytes"
	"sync"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/receipts/chain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/pkg/errors"
)

type ReceiptHandlerVersion int32

const (
	DefaultReceiptStorage  = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerChain    = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerLevelDb  = 2 //ctypes.ReceiptStorage_LEVELDB
	ReceiptHandlerLegacyV1 = 101
	ReceiptHandlerLegacyV2 = 102
	DefaultMaxReceipts     = uint64(2000)
)

func ReceiptHandlerVersionFromInt(v int32) (ReceiptHandlerVersion, error) {
	if v < 0 || v > int32(ReceiptHandlerLegacyV2) {
		return DefaultReceiptStorage, loomchain.ErrInvalidVersion
	}
	if v == 0 {
		return ReceiptHandlerChain, nil
	}
	return ReceiptHandlerVersion(v), nil
}

// ReceiptHandler implements loomchain.ReadReceiptHandler, loomchain.WriteReceiptHandler, and
// loomchain.ReceiptHandlerStore interfaces.
type ReceiptHandler struct {
	v               ReceiptHandlerVersion
	eventHandler    loomchain.EventHandler
	chainReceipts   *chain.StateDBReceipts
	leveldbReceipts *leveldb.LevelDbReceipts

	mutex         *sync.RWMutex
	receiptsCache []*types.EvmTxReceipt
	txHashList    [][]byte

	currentReceipt *types.EvmTxReceipt
}

func NewReceiptHandler(version ReceiptHandlerVersion, eventHandler loomchain.EventHandler, maxReceipts uint64) (*ReceiptHandler, error) {
	rh := &ReceiptHandler{
		v:              version,
		eventHandler:   eventHandler,
		receiptsCache:  []*types.EvmTxReceipt{},
		txHashList:     [][]byte{},
		currentReceipt: nil,
		mutex:          &sync.RWMutex{},
	}

	switch version {
	case ReceiptHandlerChain:
		rh.chainReceipts = &chain.StateDBReceipts{}
	case ReceiptHandlerLevelDb:
		leveldbHandler, err := leveldb.NewLevelDbReceipts(maxReceipts)
		if err != nil {
			return nil, errors.Wrap(err, "new leved db receipt handler")
		}
		rh.leveldbReceipts = leveldbHandler
	}
	return rh, nil
}

func (r *ReceiptHandler) Version() ReceiptHandlerVersion {
	return r.v
}

func (r *ReceiptHandler) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetReceipt(state, txHash)
	case ReceiptHandlerLevelDb:
		receipt, err := r.leveldbReceipts.GetReceipt(txHash)
		// In case receipts has been upgraded from V1 to V2 try getting the receipt from the chain.
		if err != nil {
			var chainErr error
			receipt, chainErr = r.chainReceipts.GetReceipt(state, txHash)
			if chainErr != nil {
				return receipt, errors.Wrap(chainErr, err.Error())
			}
		}
		return receipt, nil
	}
	return types.EvmTxReceipt{}, loomchain.ErrInvalidVersion
}

func (r *ReceiptHandler) GetPendingReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, receipt := range r.receiptsCache {
		if 0 == bytes.Compare(receipt.TxHash, txHash) {
			return *receipt, nil
		}
	}
	return types.EvmTxReceipt{}, errors.New("pending receipt not found")
}

func (r *ReceiptHandler) GetCurrentReceipt() *types.EvmTxReceipt {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.currentReceipt
}

func (r *ReceiptHandler) GetPendingTxHashList() [][]byte {
	r.mutex.RLock()
	hashListCopy := make([][]byte, len(r.txHashList))
	copy(hashListCopy, r.txHashList)
	r.mutex.RUnlock()
	return hashListCopy
}

func (r *ReceiptHandler) Close() error {
	switch r.v {
	case ReceiptHandlerChain:
	case ReceiptHandlerLevelDb:
		err := r.leveldbReceipts.Close()
		if err != nil {
			return errors.Wrap(err, "closing receipt leveldb")
		}
	default:
		return loomchain.ErrInvalidVersion
	}
	return nil
}

func (r *ReceiptHandler) ClearData() error {
	switch r.v {
	case ReceiptHandlerChain:
		r.chainReceipts.ClearData()
	case ReceiptHandlerLevelDb:
		r.leveldbReceipts.ClearData()
	default:
		return loomchain.ErrInvalidVersion
	}
	return nil
}

func (r *ReceiptHandler) CommitCurrentReceipt() {
	if r.currentReceipt != nil {
		r.mutex.Lock()
		r.receiptsCache = append(r.receiptsCache, r.currentReceipt)
		r.txHashList = append(r.txHashList, r.currentReceipt.TxHash)
		r.mutex.Unlock()

		r.currentReceipt = nil
	}
}

func (r *ReceiptHandler) DiscardCurrentReceipt() {
	r.currentReceipt = nil
}

func (r *ReceiptHandler) CommitBlock(state loomchain.State, height int64) error {
	var err error

	switch r.v {
	case ReceiptHandlerChain:
		r.mutex.RLock()
		err = r.chainReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
		r.mutex.RUnlock()
	case ReceiptHandlerLevelDb:
		r.mutex.RLock()
		err = r.leveldbReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
		r.mutex.RUnlock()
	default:
		err = loomchain.ErrInvalidVersion
	}

	r.mutex.Lock()
	r.txHashList = [][]byte{}
	r.receiptsCache = []*types.EvmTxReceipt{}
	r.mutex.Unlock()

	return err
}

// TODO: this doesn't need the entire state passed in, just the block header
func (r *ReceiptHandler) CacheReceipt(state loomchain.State, caller, addr loom.Address, events []*types.EventData, txErr error) ([]byte, error) {
	var status int32
	if txErr == nil {
		status = common.StatusTxSuccess
	} else {
		status = common.StatusTxFail
	}

	var err error
	var receipt types.EvmTxReceipt
	switch r.v {
	case ReceiptHandlerChain:
		r.mutex.RLock()
		receipt, err = chain.DepreciatedWriteReceipt(state.Block(), caller, addr, events, status, r.eventHandler)
		r.mutex.RUnlock()
	case ReceiptHandlerLevelDb:
		r.mutex.RLock()
		receipt, err = leveldb.WriteReceipt(state.Block(), caller, addr, events, status, r.eventHandler, int32(len(r.receiptsCache)), int64(auth.Nonce(state, caller)))
		r.mutex.RUnlock()
	default:
		err = loomchain.ErrInvalidVersion
	}

	if err != nil {
		return []byte{}, errors.Wrap(err, "receipt not written, returning empty hash")
	}
	r.currentReceipt = &receipt
	return r.currentReceipt.TxHash, err
}

func (r *ReceiptHandler) SetFailStatusCurrentReceipt() {
	if r.currentReceipt != nil {
		r.currentReceipt.Status = common.StatusTxFail
	}
}

func (r *ReceiptHandler) GetBloomFilter(height uint64) []byte {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.leveldbReceipts.GetBloomFilter(height)
}

func (r *ReceiptHandler) GetTxHashList(height uint64) ([][]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.leveldbReceipts.GetTxHashList(height)
}
