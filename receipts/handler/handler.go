package handler

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/chain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	abci "github.com/tendermint/tendermint/abci/types"
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
	if r.currentReceipt != nil && r.currentReceipt.Status == loomchain.StatusTxSuccess{
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
	block := state.Block()
	lastBlockHash :=  block.GetLastBlockID().Hash

	switch r.v {
	case ReceiptHandlerChain:
		r.mutex.RLock()
		err = r.chainReceipts.CommitBlock(state, r.receiptsCache, uint64(height), lastBlockHash)
		r.mutex.RUnlock()
	case ReceiptHandlerLevelDb:
		r.mutex.RLock()
		err = r.leveldbReceipts.CommitBlock(state, r.receiptsCache, uint64(height), lastBlockHash)
		r.mutex.RUnlock()
	default:
		err = loomchain.ErrInvalidVersion
	}

	r.mutex.Lock()
	r.txHashList = [][]byte{}
	r.receiptsCache = []*types.EvmTxReceipt{}
	r.mutex.Unlock()

	// Debug check that receitps for last blcok are consistent
	r.debugCheckTransactionIndexConsistancy(state, height)

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

func (r *ReceiptHandler) debugCheckTransactionIndexConsistancy(state loomchain.State, height int64) {
	if height > 0 {
		var resultBlockResults *ctypes.ResultBlockResults
		resultBlockResults, err := core.BlockResults(&height)
		if err != nil {
			panic(fmt.Sprintf("cannot get block results, %v", err))
		}
		deliverTxResponses := resultBlockResults.Results.DeliverTx

		resultBlock, err := core.Block(&height)
		if err != nil {
			panic(fmt.Sprintf("cannot get result block for last block, height %v ", height))
		}
		blockhash := resultBlock.BlockMeta.BlockID.Hash
		txHashList, err := common.GetTxHashList(state, uint64(height))
		if err != nil {
			panic(fmt.Sprintf("cannot get transaction hash list at height %v ", height))
		}
		r.confirmConsistancy(state, height, deliverTxResponses, txHashList, blockhash)
	}

}

func (r *ReceiptHandler) confirmConsistancy(state loomchain.State, height int64, deliverTxResponses []*abci.ResponseDeliverTx, txHashList [][]byte, blockhash []byte) {
	if height < 2 {
		return
	}
	if len(deliverTxResponses) == 0 && len(txHashList) == 0 {
		return
	}
	if len(deliverTxResponses) < len(txHashList) {
		panic(fmt.Sprintf("tendermint transaction count %v less than loom transaction count %v", len(deliverTxResponses), len(txHashList)))
	}

	tendermintEvmIndex := 0
	for index, deliverTxResponse := range deliverTxResponses {
		if deliverTxResponse.Info == utils.CallEVM || deliverTxResponse.Info == utils.DeployEvm {
			tendermintHash, err := getTxHashFromResponse(*deliverTxResponse)
			if err != nil {
				panic(fmt.Sprintf("error getting tx hash from response %v %v", deliverTxResponse, err))
			}
			if 0 != bytes.Compare(tendermintHash, txHashList[tendermintEvmIndex]) {
				panic(fmt.Sprintf("tendermint hash %v and receipt hash %v do not match at index; evm %v tendermint %v",tendermintHash, txHashList[tendermintEvmIndex], tendermintEvmIndex, index))
			}

			receipt, err := r.GetReceipt(state, txHashList[tendermintEvmIndex])
			if err != nil {
				panic(fmt.Sprintf("error getting transaction receipt %v", err))
			}
			if 0 != bytes.Compare(blockhash, receipt.BlockHash) {
				panic(fmt.Sprintf("mismatch tendermint %v and rectiopt %v block hash", blockhash, receipt.BlockHash))
			}
			if receipt.TransactionIndex != int32(tendermintEvmIndex) {
				panic(fmt.Sprintf("transaction index mismatch: txRectipt hash %v, rectipt index %v, tendermint index; evm %v tendermint %v", receipt.TxHash, receipt.TransactionIndex, tendermintEvmIndex, index))
			}
			if 0 != bytes.Compare(receipt.TxHash, txHashList[tendermintEvmIndex]) {
				panic( fmt.Sprintf("receipt database corrupt, incmpatable tx hashes %v at hash %v", receipt.TxHash, txHashList[tendermintEvmIndex] ))
			}
			tendermintEvmIndex++
		}
	}
	if tendermintEvmIndex != len(txHashList) {
		panic(fmt.Sprintf("tendermint %v loom %v transaction count mismatch", len(deliverTxResponses), tendermintEvmIndex))
	}
}

func getTxHashFromResponse(response abci.ResponseDeliverTx) ([]byte, error) {
	switch response.Info {
	case utils.DeployEvm:
		{
			dr := vm.DeployResponse{}
			if err := proto.Unmarshal(response.Data, &dr); err != nil {
				return nil, errors.Wrapf(err, "deploy resonse does not unmarshal")
			}
			drd := vm.DeployResponseData{}
			if err := proto.Unmarshal(dr.Output, &drd); err != nil {
				return nil, errors.Wrap(err, "deploy response data does not unmarshal")
			}
			return drd.TxHash, nil
		}
	case utils.CallEVM:
		return response.Data, nil
	default:
		return nil, nil
	}
}










































/**/
