package loom

import (
	"context"

	abci "github.com/tendermint/abci/types"

	"github.com/loomnetwork/test-sdk/store"
)

type State interface {
	// Get returns nil iff key doesn't exist. Panics on nil key.
	Get(key []byte) []byte

	// Set sets the key. Panics on nil key.
	Set(key, value []byte)

	// Delete deletes the key. Panics on nil key.
	Delete(key []byte)

	Block() abci.Header
	Context() context.Context
	WithContext(ctx context.Context) State
}

type simpleState struct {
	store store.KVStore
	block abci.Header

	ctx context.Context
}

var _ = State(&simpleState{})

func (s *simpleState) Get(key []byte) []byte {
	return s.store.Get(key)
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
	Handle(state State, txBytes []byte) error
}

type TxHandlerFunc func(state State, txBytes []byte) error

func (f TxHandlerFunc) Handle(state State, txBytes []byte) error {
	return f(state, txBytes)
}

type QueryHandler interface {
	Handle(state State, path string, data []byte) ([]byte, error)
}

type Application struct {
	abci.BaseApplication

	curBlockHeader abci.Header

	TxHandler
	QueryHandler
	Store store.CommitKVStore
}

var _ abci.Application = &Application{}

func (a *Application) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	a.curBlockHeader = req.Header
	return abci.ResponseBeginBlock{}
}

func (a *Application) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	if len(txBytes) == 0 {
		return abci.ResponseCheckTx{Code: 1, Log: "transaction empty"}
	}
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	state := &simpleState{
		store: a.Store,
		block: a.curBlockHeader,
		ctx:   context.Background(),
	}
	err := a.TxHandler.Handle(state, txBytes)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK}
}

func (a *Application) Commit() abci.ResponseCommit {
	a.Store.Commit()
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

func (a *Application) State() State {
	return &simpleState{
		store: a.Store,
		block: a.curBlockHeader,
		ctx:   context.Background(),
	}
}
