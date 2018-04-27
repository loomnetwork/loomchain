package loomchain

import (
	"context"

	abci "github.com/tendermint/abci/types"
	common "github.com/tendermint/tmlibs/common"

	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loom/store"
)

type ReadOnlyState interface {
	store.KVReader
	Block() types.BlockHeader
}

type State interface {
	ReadOnlyState
	store.KVWriter
	Context() context.Context
	WithContext(ctx context.Context) State
}

type StoreState struct {
	ctx   context.Context
	store store.KVStore
	block types.BlockHeader
}

var _ = State(&StoreState{})

func blockHeaderFromAbciHeader(header *abci.Header) types.BlockHeader {
	return types.BlockHeader{
		ChainID: header.ChainID,
		Height:  header.Height,
		Time:    header.Time,
		NumTxs:  header.NumTxs,
		LastBlockID: types.BlockID{
			Hash: header.LastBlockID.Hash,
			Parts: types.PartSetHeader{
				Total: header.LastBlockID.Parts.Total,
				Hash:  header.LastBlockID.Parts.Hash,
			},
		},
		LastCommitHash: header.LastCommitHash,
		DataHash:       header.DataHash,
		ValidatorsHash: header.ValidatorsHash,
		AppHash:        header.AppHash,
	}
}

func NewStoreState(ctx context.Context, store store.KVStore, block abci.Header) *StoreState {
	return &StoreState{
		ctx:   ctx,
		store: store,
		block: blockHeaderFromAbciHeader(&block),
	}
}

func (s *StoreState) Get(key []byte) []byte {
	return s.store.Get(key)
}

func (s *StoreState) Has(key []byte) bool {
	return s.store.Has(key)
}

func (s *StoreState) Set(key, value []byte) {
	s.store.Set(key, value)
}

func (s *StoreState) Delete(key []byte) {
	s.store.Delete(key)
}

func (s *StoreState) Block() types.BlockHeader {
	return s.block
}

func (s *StoreState) Context() context.Context {
	return s.ctx
}

func (s *StoreState) WithContext(ctx context.Context) State {
	return &StoreState{
		store: s.store,
		block: s.block,
		ctx:   ctx,
	}
}

func StateWithPrefix(prefix []byte, state State) State {
	return &StoreState{
		store: store.PrefixKVStore(prefix, state),
		block: state.Block(),
		ctx:   state.Context(),
	}
}

type TxHandler interface {
	ProcessTx(state State, txBytes []byte) (TxHandlerResult, error)
}

type TxHandlerFunc func(state State, txBytes []byte) (TxHandlerResult, error)

type TxHandlerResult struct {
	Data []byte
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tmlibs/pubsub/query)
	Tags []common.KVPair
}

func (f TxHandlerFunc) ProcessTx(state State, txBytes []byte) (TxHandlerResult, error) {
	return f(state, txBytes)
}

type QueryHandler interface {
	Handle(state ReadOnlyState, path string, data []byte) ([]byte, error)
}

type Application struct {
	lastBlockHeader abci.Header
	curBlockHeader  abci.Header

	Store store.VersionedKVStore
	Init  func(State) error
	TxHandler
	QueryHandler
}

var _ abci.Application = &Application{}

func (a *Application) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{
		LastBlockAppHash: a.Store.Hash(),
		LastBlockHeight:  a.Store.Version(),
	}
}

func (a *Application) SetOption(req abci.RequestSetOption) abci.ResponseSetOption {
	return abci.ResponseSetOption{}
}

func (a *Application) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	if a.height() != 1 {
		panic("state version is not 1")
	}

	state := NewStoreState(
		context.Background(),
		a.Store,
		abci.Header{},
	)

	if a.Init != nil {
		err := a.Init(state)
		if err != nil {
			panic(err)
		}
	}
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
	_, err := a.processTx(txBytes, true)
	if err != nil {
		return abci.ResponseCheckTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	r, err := a.processTx(txBytes, false)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Data: r.Data, Tags: r.Tags}
}

func (a *Application) processTx(txBytes []byte, fake bool) (TxHandlerResult, error) {
	storeTx := store.WrapAtomic(a.Store).BeginTx()
	// This is a noop if committed
	defer storeTx.Rollback()

	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
	)
	r, err := a.TxHandler.ProcessTx(state, txBytes)
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
	appHash, _, err := a.Store.SaveVersion()
	if err != nil {
		panic(err)
	}
	a.lastBlockHeader = a.curBlockHeader
	return abci.ResponseCommit{
		Data: appHash,
	}
}

func (a *Application) Query(req abci.RequestQuery) abci.ResponseQuery {
	if a.QueryHandler == nil {
		return abci.ResponseQuery{Code: 1, Log: "not implemented"}
	}

	result, err := a.QueryHandler.Handle(a.ReadOnlyState(), req.Path, req.Data)
	if err != nil {
		return abci.ResponseQuery{Code: 1, Log: err.Error()}
	}

	return abci.ResponseQuery{Code: abci.CodeTypeOK, Value: result}
}

func (a *Application) height() int64 {
	return a.Store.Version() + 1
}

func (a *Application) ReadOnlyState() State {
	return NewStoreState(
		nil,
		a.Store,
		a.lastBlockHeader,
	)
}
