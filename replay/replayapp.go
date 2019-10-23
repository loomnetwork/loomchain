package replay

import (
	"context"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler"
)

type ReplayApplication interface {
	abci.Application
	SetTracer(vm.Tracer) error
}

var _ abci.Application = (*replayApplication)(nil)
var _ abci.Application = ReplayApplication(nil)

type replayApplication struct {
	height           int64
	blockstore       store.BlockStore
	store            store.VersionedKVStore
	receiptsVersion  int32
	getValidatorSet  state.GetValidatorSet
	config           *chainconfig.Config
	tracer           vm.Tracer
	txHandlerFactory txhandler.TxHandlerFactory
	txHandler        txhandler.TxHandler
}

func NewReplayApplication(
	height int64,
	blockstore store.BlockStore,
	store store.VersionedKVStore,
	receiptsVersion int32,
	getValidatorSet state.GetValidatorSet,
	config *chainconfig.Config,
	txHandlerFactory txhandler.TxHandlerFactory,
	tracer vm.Tracer,
) (*replayApplication, error) {
	app := &replayApplication{
		height:           height,
		blockstore:       blockstore,
		store:            store,
		receiptsVersion:  receiptsVersion,
		getValidatorSet:  getValidatorSet,
		config:           config,
		txHandlerFactory: txHandlerFactory,
	}
	if err := app.SetTracer(tracer); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *replayApplication) SetTracer(tracer vm.Tracer) error {
	newTxHandle, err := a.txHandlerFactory.TxHandler(tracer, false)
	if err != nil {
		return errors.Wrap(err, "making transaction handle")
	}
	a.tracer = tracer
	a.txHandler = newTxHandle
	return nil
}

func (a *replayApplication) DeliverTx(tx []byte) abci.ResponseDeliverTx {
	splitStoreTx := store.WrapAtomic(a.store).BeginTx()
	defer splitStoreTx.Rollback()
	resultBlock, err := a.blockstore.GetBlockByHeight(&a.height)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}

	storeState := state.NewStoreState2(
		context.Background(),
		splitStoreTx,
		blockHeaderFromHeader(resultBlock.BlockMeta.Header),
		a.getValidatorSet,
	).WithOnChainConfig(a.config)

	r, err := a.txHandler.ProcessTx(storeState, tx, false)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Data: r.Data, Tags: r.Tags, Info: r.Info}
}

func (_ *replayApplication) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return abci.ResponseBeginBlock{}
}

func (a *replayApplication) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	a.height++
	return abci.ResponseEndBlock{}
}

func (_ *replayApplication) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	return abci.ResponseInitChain{}
}

func (_ *replayApplication) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{}
}

func (_ *replayApplication) SetOption(req abci.RequestSetOption) abci.ResponseSetOption {
	return abci.ResponseSetOption{}
}

func (_ *replayApplication) CheckTx(tx []byte) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{Code: 0}
}

func (_ *replayApplication) Commit() abci.ResponseCommit {
	return abci.ResponseCommit{}
}

func (_ *replayApplication) Query(req abci.RequestQuery) abci.ResponseQuery {
	return abci.ResponseQuery{Code: 0}
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
