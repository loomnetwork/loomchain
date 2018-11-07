package handler

import (
	"bytes"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/receipts/chain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type ReceiptHandlerVersion int32

const (
	DefaultReceiptStorage = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerChain   = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerLevelDb = 2 //ctypes.ReceiptStorage_LEVELDB
	HashLength            = 32
	DefaultMaxReceipts    = uint64(2000)
)

func ReceiptHandlerVersionFromInt(v int32) (ReceiptHandlerVersion, error) {
	if v < 0 || v > int32(ReceiptHandlerLevelDb) {
		return DefaultReceiptStorage, loomchain.ErrInvalidVersion
	}
	if v == 0 {
		return ReceiptHandlerChain, nil
	}
	return ReceiptHandlerVersion(v), nil
}

//Allows runtime swapping of receipt handlers
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

func (r *ReceiptHandler) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetReceipt(state, txHash)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.GetReceipt(txHash)
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

func (r *ReceiptHandler) GetCurrentReceipt(txHash []byte) (*types.EvmTxReceipt, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.currentReceipt, nil
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

func (r *ReceiptHandler) ReadOnlyHandler() loomchain.ReadReceiptHandler {
	return r
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

func (r *ReceiptHandler) CacheReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, txErr error) ([]byte, error) {
	var status int32
	if txErr == nil {
		status = loomchain.StatusTxSuccess
	} else {
		status = loomchain.StatusTxFail
	}
	receipt, err := common.WriteReceipt(state, caller, addr, events, status, r.eventHandler)
	if err != nil {
		errors.Wrap(err, "receipt not written, returning empty hash")
		return []byte{}, err
	}
	r.currentReceipt = &receipt
	return r.currentReceipt.TxHash, err
}

func (r *ReceiptHandler) SetFailStatusCurrentReceipt() {
	if r.currentReceipt != nil {
		r.currentReceipt.Status = loomchain.StatusTxFail
	}
}

func (r *ReceiptHandler) updateReceipt(state loomchain.State, receipt types.EvmTxReceipt) error {
	var err error

	switch r.v {
	case ReceiptHandlerChain:
		r.mutex.RLock()
		err = r.chainReceipts.UpdateReceipt(state, receipt)
		r.mutex.RUnlock()
	case ReceiptHandlerLevelDb:
		r.mutex.RLock()
		err = r.leveldbReceipts.UpdateReceipt(receipt)
		r.mutex.RUnlock()
	default:
		err = loomchain.ErrInvalidVersion
	}
	return err
}

func (r *ReceiptHandler) UpdateLastBlock(state loomchain.State, height int64) error {
	if height > 0 {
		var resultBlockResults *ctypes.ResultBlockResults
		resultBlockResults, err := core.BlockResults(&height)
		if err != nil {
			return  errors.Wrapf(err, "cannot get result block results for last block, height %v ", height)
		}
		txs := resultBlockResults.Results.DeliverTx
		if len(txs) == 0 {
			return nil
		}

		var resultBlock *ctypes.ResultBlock
		resultBlock, err = core.Block(&height)
		if err != nil {
			return  errors.Wrapf(err, "cannot get result block for last block, height %v ", height)
		}
		blockHash := resultBlock.BlockMeta.BlockID.Hash

		numEvmTxs := 0
		for _, deliverTx := range txs {
			if deliverTx.Info == utils.CallEVM || deliverTx.Info == utils.DeployEvm {
				numEvmTxs++
				var txHash []byte
				if deliverTx.Info == utils.DeployEvm {
					dr := vm.DeployResponse{}
					if err := proto.Unmarshal(deliverTx.Data, &dr); err != nil {
						log.Error("deploy resonse does not unmarshal")
						continue
					}
					drd := vm.DeployResponseData{}
					if err := proto.Unmarshal(dr.Output, &drd); err != nil {
						log.Error("deploy response data does not unmarshal")
						continue
					}
					txHash = drd.TxHash
				} else {
					txHash = deliverTx.Data
				}
				receipt, err := r.GetReceipt(state, txHash)
				if err != nil {
					log.Error( "error %v getting transaction receipt", err)
					continue
				}
				receipt.BlockHash = blockHash
				receipt.TransactionIndex = int32(numEvmTxs-1)
				if err := r.updateReceipt(state, receipt); err != nil {
					log.Error("error %v updating receipt", err)
					continue
				}
			}
		}
	}
	return nil
}