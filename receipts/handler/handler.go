package handler

import (
	"sync"
	
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/pkg/errors"
	
	// todo	"github.com/loomnetwork/loomchain/builtin/plugins/config"
	
	"github.com/loomnetwork/loomchain/receipts/chain"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
)

type ReceiptHandlerVersion int32

const (
	DefaultReceiptStorage = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerChain   = 1 //ctypes.ReceiptStorage_CHAIN
	ReceiptHandlerLevelDb = 2 //ctypes.ReceiptStorage_LEVELDB
	HashLength            = 32
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
	
	receiptsMutex   *sync.RWMutex
	txHashMutex      *sync.RWMutex
	receiptsCache   []*types.EvmTxReceipt
	txHashList      [][]byte
	
	currentReceipt  *types.EvmTxReceipt
}

func  NewReceiptHandler(version ReceiptHandlerVersion,	eventHandler loomchain.EventHandler) (*ReceiptHandler, error) {
	rh := &ReceiptHandler{
		v:               version,
		eventHandler:    eventHandler,
		receiptsCache:   []*types.EvmTxReceipt{},
		txHashList:      [][]byte{},
		currentReceipt:  nil,
		receiptsMutex:   &sync.RWMutex{},
		txHashMutex:     &sync.RWMutex{},
	}
	
	switch version {
	case ReceiptHandlerChain:
		rh.chainReceipts = &chain.StateDBReceipts{}
	case ReceiptHandlerLevelDb:
		leveldbHandler, err := leveldb.NewLevelDbReceipts(leveldb.Default_DBHeight)
		if err != nil  {
			return nil, errors.Wrap(err,"new leved db receipt handler")
		}
		rh.leveldbReceipts = leveldbHandler
	}
	return rh, nil
}

/*
//todo figure out how to reintegrate the configs from blockchain
func TODOGetConfigState() {
	var configContractAddress loom.Address
	return func(s loomchain.State) (loomchain.WriteReceiptHandler, error) {
		if (0 == configContractAddress.Compare(loom.Address{})) {
			var err error
			configContractAddress, err = common.GetConfigContractAddress(s, createRegistry)
			if err != nil {
				return nil, errors.Wrap(err, "config contract address")
			}
		}
		configState := common.GetConfignState(s, configContractAddress)
		protoValue := configState.Get(config.StateKey(config.ConfigKeyRecieptStrage))
		value := ctypes.Value{}
		if err := proto.Unmarshal(protoValue, &value); err != nil {
			return nil, errors.Wrap(err ,"unmarshal config value")
		}
		switch value.GetReceiptStorage() {
			case ctypes.ReceiptStorage_CHAIN: return &chain.WriteStateReceipts{s}, nil
			case ctypes.ReceiptStorage_LEVELDB:	return &leveldb.WriteLevelDbReceipts{s}, nil
			default: return nil, errors.Errorf("unrecognised receipt storage method, %v", value.GetReceiptStorage())
		}
	}
}
*/

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
	found := false
	var index int
	r.txHashMutex.RLock()
	for i := 0 ; i<len(r.txHashList) && !found ; i++ {
		if string(r.txHashList[i]) == string(txHash) {
			found = true
			index = i
		}
	}
	r.txHashMutex.RUnlock()
	if !found {
		return types.EvmTxReceipt{}, errors.New("pending receipt not found")
	}
	r.receiptsMutex.RLock()
	receipt := *r.receiptsCache[index]
	r.receiptsMutex.RUnlock()
	return receipt, nil
}

func (r *ReceiptHandler) GetPendingTxHashList() ([][]byte) {
	r.txHashMutex.RLock()
	hashListCopy := r.txHashList
	r.txHashMutex.RUnlock()
	return hashListCopy
}

func (r *ReceiptHandler) Close() (error) {
	switch r.v {
	case ReceiptHandlerChain:
	case ReceiptHandlerLevelDb:
		err:= r.leveldbReceipts.Close()
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

func (r *ReceiptHandler) ReadOnlyHandler() loomchain.ReadReceiptHandler{
	return r
}

func (r *ReceiptHandler) CommitCurrentReceipt()  {
	if  r.currentReceipt != nil {
		r.receiptsMutex.Lock()
		r.txHashMutex.Lock()
		r.receiptsCache = append(r.receiptsCache, r.currentReceipt)
		r.txHashList = append(r.txHashList, r.currentReceipt.TxHash)
		r.txHashMutex.Unlock()
		r.receiptsMutex.Unlock()
		
		r.currentReceipt = nil
	}
}

func (r *ReceiptHandler) CommitBlock(state loomchain.State, height int64) error {
	var err error
	
	switch r.v {
	case ReceiptHandlerChain:
		r.receiptsMutex.RLock()
		err = r.chainReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
		r.receiptsMutex.RUnlock()
	case ReceiptHandlerLevelDb:
		r.receiptsMutex.RLock()
		err = r.leveldbReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
		r.receiptsMutex.RUnlock()
	default:
		err = loomchain.ErrInvalidVersion
	}
	
	r.txHashMutex.Lock()
	r.receiptsMutex.Lock()
	r.txHashList = [][]byte{}
	r.receiptsCache = []*types.EvmTxReceipt{}
	r.receiptsMutex.Unlock()
	r.txHashMutex.Unlock()
	
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

