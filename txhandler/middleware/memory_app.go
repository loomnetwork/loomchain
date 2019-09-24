package middleware

import (
	"context"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler"
)

type InMemoryApp interface {
	ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error)
	TraceProcessTx(txBytes []byte, traceCfg eth.TraceConfig) (txhandler.TxHandlerResult, vm.Tracer, error)
}

type inMemoryApp struct {
	lastBlockHeader  abci.Header
	curBlockHeader   abci.Header
	curBlockHash     []byte
	store            store.VersionedKVStore
	txHandler        txhandler.TxHandler
	receiptsVersion  int32
	getValidatorSet  state.GetValidatorSet
	config           *chainconfig.Config
	txHandlerFactory txhandler.TxHandlerFactory
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
	txHandlerFactory txhandler.TxHandlerFactory,
) InMemoryApp {
	return &inMemoryApp{
		lastBlockHeader:  lastBlockHeader,
		curBlockHeader:   curBlockHeader,
		curBlockHash:     curBlockHash,
		store:            store,
		txHandler:        txHandler,
		receiptsVersion:  receiptsVersion,
		getValidatorSet:  getValidatorSet,
		config:           config,
		txHandlerFactory: txHandlerFactory,
	}
}

func (ma *inMemoryApp) ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error) {
	return ma.processTx(txBytes, ma.txHandler)
}

func (ma *inMemoryApp) processTx(txBytes []byte, txHandler txhandler.TxHandler) (txhandler.TxHandlerResult, error) {
	splitStoreTx := store.WrapAtomic(ma.store).BeginTx()
	defer splitStoreTx.Rollback()
	storeState := state.NewStoreState(
		context.Background(),
		splitStoreTx,
		ma.curBlockHeader,
		ma.curBlockHash,
		ma.getValidatorSet,
	).WithOnChainConfig(ma.config)
	return txHandler.ProcessTx(storeState, txBytes, false)
}

func (ma *inMemoryApp) TraceProcessTx(txBytes []byte, traceCfg eth.TraceConfig) (txhandler.TxHandlerResult, vm.Tracer, error) {
	tracer, err := createTracer(traceCfg)
	if err != nil {
		return txhandler.TxHandlerResult{}, nil, err
	}
	// Pass pointer to tracer as we want evm to run with this object, not a copy
	traceTxHandle, err := ma.txHandlerFactory.TxHandler(tracer)
	if err != nil {
		return txhandler.TxHandlerResult{}, nil, err
	}
	res, err := ma.processTx(txBytes, traceTxHandle)
	return res, tracer, err
}

func createTracer(traceCfg eth.TraceConfig) (vm.Tracer, error) {
	if traceCfg.Tracer == nil {
		return vm.NewStructLogger(traceCfg.LogConfig), nil
	}
	tracer, err := tracers.New(*traceCfg.Tracer)
	if err != nil {
		return nil, err
	}
	return tracer, nil
}
