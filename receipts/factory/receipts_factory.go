package factory

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	//todo	"github.com/loomnetwork/loomchain/builtin/plugins/config"

	"github.com/loomnetwork/loomchain/receipts"
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
		return 0, receipts.ErrInvalidVersion
	}
	if v == 0 {
		return ReceiptHandlerChain, nil
	}
	return ReceiptHandlerVersion(v), nil
}

//Allows runtime swapping of receipt handlers
type ReceiptHandlerFactory struct {
	v               ReceiptHandlerVersion
	chainReceipts   *chain.WriteStateReceipts
	leveldbReceipts *leveldb.WriteLevelDbReceipts
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

func (r *ReceiptHandlerFactory) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetReceipt(state, txHash)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.GetReceipt(state, txHash)
	}
	return types.EvmTxReceipt{}, receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) Close() {
	switch r.v {
	case ReceiptHandlerChain:
		r.chainReceipts.Close()
	case ReceiptHandlerLevelDb:
		r.leveldbReceipts.Close()
	}
}

func (r *ReceiptHandlerFactory) ClearData() error {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.ClearData()
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.ClearData()
	}
	return receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) SaveEventsAndHashReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.SaveEventsAndHashReceipt(state, caller, addr, events, err)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.SaveEventsAndHashReceipt(state, caller, addr, events, err)
	}
	return nil, receipts.ErrInvalidVersion
}

func NewReceiptHandlerFactory(v ReceiptHandlerVersion, eh loomchain.EventHandler) (receipts.ReceiptHandler, error) {
	r := &ReceiptHandlerFactory{v: v}
	switch r.v {
	case ReceiptHandlerChain:
		wsr := &chain.WriteStateReceipts{eh}
		r.chainReceipts = wsr
		return r, nil
	case ReceiptHandlerLevelDb:
		ldbr, err := leveldb.NewWriteLevelDbReceipts(eh)
		r.leveldbReceipts = ldbr
		return r, err
	}
	return nil, receipts.ErrInvalidVersion
}
