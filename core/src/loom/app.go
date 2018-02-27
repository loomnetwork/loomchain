package loom

import (
	"context"

	abci "github.com/tendermint/abci/types"
	common "github.com/tendermint/tmlibs/common"

	"loom/store"
)

type ReadOnlyState interface {
	store.KVReader
	Block() abci.Header
}

type State interface {
	ReadOnlyState
	store.KVWriter
	Context() context.Context
	WithContext(ctx context.Context) State
}

type simpleState struct {
	store store.KVStore
	block abci.Header
	ctx   context.Context
}

var _ = State(&simpleState{})

func (s *simpleState) Get(key []byte) []byte {
	return s.store.Get(key)
}

func (s *simpleState) Has(key []byte) bool {
	return s.store.Has(key)
}

func (s *simpleState) Set(key, value []byte) {
	s.store.Set(key, value)
}

func (s *simpleState) Delete(key []byte) {
	s.store.Delete(key)
}

func (s *simpleState) Block() abci.Header {
	return s.block
}

func (s *simpleState) Context() context.Context {
	return s.ctx
}

func (s *simpleState) WithContext(ctx context.Context) State {
	return &simpleState{
		store: s.store,
		block: s.block,
		ctx:   ctx,
	}
}

type TxHandler interface {
	Handle(state State, txBytes []byte) (TxHandlerResult, error)
}

type TxHandlerResult struct {
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tmlibs/pubsub/query)
	Tags []common.KVPair
}

type TxHandlerFunc func(state State, txBytes []byte) (TxHandlerResult, error)

func (f TxHandlerFunc) Handle(state State, txBytes []byte) (TxHandlerResult, error) {
	return f(state, txBytes)
}

type QueryHandler interface {
	Handle(state ReadOnlyState, path string, data []byte) ([]byte, error)
}

type Application struct {
	lastBlockHeader abci.Header
	curBlockHeader  abci.Header

	Store store.VersionedKVStore
	TxHandler
	QueryHandler
}

var _ abci.Application = &Application{}

func (a *Application) Info(req abci.RequestInfo) abci.ResponseInfo {
	println(a.Store.Version())
	return abci.ResponseInfo{
		LastBlockAppHash: a.Store.Hash(),
		LastBlockHeight:  a.Store.Version(),
	}
}

func (a *Application) SetOption(req abci.RequestSetOption) abci.ResponseSetOption {
	return abci.ResponseSetOption{}
}

func (a *Application) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	return abci.ResponseInitChain{}
}

func (a *Application) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	block := req.Header
	if block.Height != a.height() {
		panic("state version does not match begin block height")
	}
	a.curBlockHeader = block
	return abci.ResponseBeginBlock{}
}

func (a *Application) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	if req.Height != a.height() {
		panic("state version does not match end block height")
	}
	return abci.ResponseEndBlock{}
}

func (a *Application) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	_, err := a.runTx(txBytes, true)
	if err != nil {
		return abci.ResponseCheckTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	r, err := a.runTx(txBytes, false)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Tags: r.Tags}
}

func (a *Application) runTx(txBytes []byte, fake bool) (TxHandlerResult, error) {
	storeTx := store.WrapAtomic(a.Store).BeginTx()
	// This is a noop if committed
	defer storeTx.Rollback()

	state := &simpleState{
		store: storeTx,
		block: a.curBlockHeader,
		ctx:   context.Background(),
	}
	r, err := a.TxHandler.Handle(state, txBytes)
	if err != nil {
		return r, err
	}
	if !fake {
		storeTx.Commit()
	}
	return r, nil
}

// Commit commits the current block
func (a *Application) Commit() abci.ResponseCommit {
	_, err := a.Store.SaveVersion()
	if err != nil {
		panic(err)
	}
	a.lastBlockHeader = a.curBlockHeader
	return abci.ResponseCommit{}
}

func (a *Application) Query(req abci.RequestQuery) abci.ResponseQuery {
	if a.QueryHandler == nil {
		return abci.ResponseQuery{Code: 1, Log: "not implemented"}
	}

	result, err := a.QueryHandler.Handle(a.State(), req.Path, req.Data)
	if err != nil {
		return abci.ResponseQuery{Code: 1, Log: err.Error()}
	}

	return abci.ResponseQuery{Code: abci.CodeTypeOK, Value: result}
}

func (a *Application) height() int64 {
	return a.Store.Version() + 1
}

func (a *Application) State() ReadOnlyState {
	return &simpleState{
		store: a.Store,
		block: a.lastBlockHeader,
	}
}
