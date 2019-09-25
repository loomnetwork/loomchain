package middleware

import (
	"context"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler"
)

type InMemoryApp interface {
	ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error)
	TraceProcessTx(txBytes []byte, traceCfg eth.TraceConfig) (txhandler.TxHandlerResult, vm.Tracer, error)
	NextBlock()
	Height() int64
}

type inMemoryApp struct {
	height           int64
	blockstore       store.BlockStore
	store            store.VersionedKVStore
	txHandler        txhandler.TxHandler
	receiptsVersion  int32
	getValidatorSet  state.GetValidatorSet
	config           *chainconfig.Config
	txHandlerFactory txhandler.TxHandlerFactory
}

func NewInMemoryApp(
	height int64,
	blockstore store.BlockStore,
	store store.VersionedKVStore,
	txHandler txhandler.TxHandler,
	receiptsVersion int32,
	getValidatorSet state.GetValidatorSet,
	config *chainconfig.Config,
	txHandlerFactory txhandler.TxHandlerFactory,
) InMemoryApp {
	return &inMemoryApp{
		height:           height,
		blockstore:       blockstore,
		store:            store,
		txHandler:        txHandler,
		receiptsVersion:  receiptsVersion,
		getValidatorSet:  getValidatorSet,
		config:           config,
		txHandlerFactory: txHandlerFactory,
	}
}

func (ma *inMemoryApp) NextBlock() {
	ma.height++
}

func (ma *inMemoryApp) Height() int64 {
	return ma.height
}

func (ma *inMemoryApp) ProcessTx(txBytes []byte) (txhandler.TxHandlerResult, error) {
	return ma.processTx(txBytes, ma.txHandler)
}

func (ma *inMemoryApp) processTx(txBytes []byte, txHandler txhandler.TxHandler) (txhandler.TxHandlerResult, error) {
	splitStoreTx := store.WrapAtomic(ma.store).BeginTx()
	defer splitStoreTx.Rollback()
	resultBlock, err := ma.blockstore.GetBlockByHeight(&ma.height)
	if err != nil {
		return txhandler.TxHandlerResult{}, errors.Errorf("retriving block for height %d", ma.height)
	}

	storeState := state.NewStoreState2(
		context.Background(),
		splitStoreTx,
		blockHeaderFromHeader(resultBlock.BlockMeta.Header),
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
	traceTxHandle, err := ma.txHandlerFactory.TxHandler(tracer, false)
	if err != nil {
		return txhandler.TxHandlerResult{}, nil, err
	}
	res, err := ma.processTx(txBytes, traceTxHandle)
	return res, tracer, err
}

func createTracer(traceCfg eth.TraceConfig) (vm.Tracer, error) {
	if traceCfg.Tracer == nil || len(*traceCfg.Tracer) == 0 {
		return vm.NewStructLogger(traceCfg.LogConfig), nil
	}
	tracer, err := tracers.New(*traceCfg.Tracer)
	if err != nil {
		return nil, err
	}
	return tracer, nil
}

func blockHeaderFromHeader(header ttypes.Header) types.BlockHeader {
	return types.BlockHeader{
		ChainID: header.ChainID,
		Height:  header.Height,
		Time:    int64(header.Time.Unix()),
		NumTxs:  int32(header.NumTxs),
		LastBlockID: types.BlockID{
			Hash: header.LastBlockID.Hash,
		},
		ValidatorsHash: header.ValidatorsHash,
		AppHash:        header.AppHash,
	}
}
