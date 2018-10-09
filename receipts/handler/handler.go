package handler

import (
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
	
	receiptsCache   []*types.EvmTxReceipt
}

func  NewReceiptHandler(version ReceiptHandlerVersion,	eventHandler loomchain.EventHandler) *ReceiptHandler {
	return &ReceiptHandler{
		v:               version,
		eventHandler:    eventHandler,
		chainReceipts:   &chain.StateDBReceipts{},
		leveldbReceipts: &leveldb.LevelDbReceipts{MaxDbSize: leveldb.Default_DBHeight},
		receiptsCache:   []*types.EvmTxReceipt{},
	}
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


func (r *ReceiptHandler) CommitBlock(state loomchain.State, height int64) error {
	var err error
	switch r.v {
	case ReceiptHandlerChain:
		err = r.chainReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
	case ReceiptHandlerLevelDb:
		r.leveldbReceipts.CommitBlock(state, r.receiptsCache, uint64(height))
	default:
		err = loomchain.ErrInvalidVersion
	}
	r.receiptsCache = []*types.EvmTxReceipt{}
	return err
}

func (r *ReceiptHandler) CacheReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, txErr error) ([]byte, error) {
	var status int32
	if txErr == nil {
		status = loomchain.StatusTxSuccess
	} else {
		status = loomchain.StatusTxFail
	}
	txReceipt, err := common.WriteReceipt(state, caller, addr, events, status, r.eventHandler)
	if err != nil {
		errors.Wrap(err, "receipt not written")
	}
	r.receiptsCache = append(r.receiptsCache, &txReceipt)
	return txReceipt.TxHash, err
}

func (r *ReceiptHandler) SetFailStatusCurrentReceipt() {
	if len(r.receiptsCache) > 0 {
		if r.receiptsCache[len(r.receiptsCache)-1] != nil {
			r.receiptsCache[len(r.receiptsCache)-1].Status = loomchain.StatusTxFail
		}
	}
}

