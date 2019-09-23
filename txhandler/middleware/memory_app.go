package middleware

import (
	"context"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler"
)

type InMemoryApp interface {
	ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error)
	TraceProcessTx(txBytes []byte, traceCfg eth.TraceConfig) (txhandler.TxHandlerResult, error)
}

type inMemoryApp struct {
	LastBlockHeader abci.Header
	CurBlockHeader  abci.Header
	CurBlockHash    []byte
	Store           store.VersionedKVStore
	TxHandler       txhandler.TxHandler
	ReceiptsVersion int32
	GetValidatorSet state.GetValidatorSet
	config          *chainconfig.Config
}

func NewInMemoryApp(
	lastBlockHeader abci.Header,
	curBlockHeader abci.Header,
	curBlockHash []byte,
	store store.VersionedKVStore,
	txHandler txhandler.TxHandler,
	receiptsVersion int32,
	getValidatorSet state.GetValidatorSet,
	config *chainconfig.Config,
) InMemoryApp {
	return &inMemoryApp{
		LastBlockHeader: lastBlockHeader,
		CurBlockHeader:  curBlockHeader,
		CurBlockHash:    curBlockHash,
		Store:           store,
		TxHandler:       txHandler,
		ReceiptsVersion: receiptsVersion,
		GetValidatorSet: getValidatorSet,
		config:          config,
	}
}

func (ma *inMemoryApp) ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error) {
	return ma.processTx(txBytes, ma.TxHandler)
}

func (ma *inMemoryApp) processTx(txBytes []byte, txHandler txhandler.TxHandler) (txhandler.TxHandlerResult, error) {
	splitStoreTx := store.WrapAtomic(ma.Store).BeginTx()
	defer splitStoreTx.Rollback()
	storeState := state.NewStoreState(
		context.Background(),
		splitStoreTx,
		ma.CurBlockHeader,
		ma.CurBlockHash,
		ma.GetValidatorSet,
	).WithOnChainConfig(ma.config)
	return txHandler.ProcessTx(storeState, txBytes, false)
}

func (ma *inMemoryApp) TraceProcessTx(txBytes []byte, traceCfg eth.TraceConfig) (txhandler.TxHandlerResult, error) {
	traceTxHandle, err := TraceTxHandle(traceCfg)
	if err != nil {
		return txhandler.TxHandlerResult{}, err
	}
	return ma.processTx(txBytes, traceTxHandle)
}

func TraceTxHandle(traceCfg eth.TraceConfig) (txhandler.TxHandler, error) {
	return nil, nil
}
