package loomchain

import (
	"context"
	"fmt"
	"time"

	"github.com/loomnetwork/loomchain/eth/utils"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
)

type ReadOnlyState interface {
	store.KVReader
	Validators() []*loom.Validator
	Block() types.BlockHeader
}

type State interface {
	ReadOnlyState
	store.KVWriter
	SetValidatorPower(pubKey []byte, power int64)
	Context() context.Context
	WithContext(ctx context.Context) State
}

type StoreState struct {
	ctx        context.Context
	store      store.KVStore
	block      types.BlockHeader
	validators loom.ValidatorSet
}

var _ = State(&StoreState{})

func blockHeaderFromAbciHeader(header *abci.Header) types.BlockHeader {
	return types.BlockHeader{
		ChainID: header.ChainID,
		Height:  header.Height,
		Time:    header.Time,
		NumTxs:  header.NumTxs,
		LastBlockID: types.BlockID{
			Hash: header.LastBlockHash,
		},
		ValidatorsHash: header.ValidatorsHash,
		AppHash:        header.AppHash,
	}
}

func NewStoreState(ctx context.Context, store store.KVStore, block abci.Header, curBlockHash []byte) *StoreState {
	blockHeader := blockHeaderFromAbciHeader(&block)
	blockHeader.CurrentHash = curBlockHash
	return &StoreState{
		ctx:        ctx,
		store:      store,
		block:      blockHeader,
		validators: loom.NewValidatorSet(),
	}
}

func (c *StoreState) Range(prefix []byte) plugin.RangeData {
	return c.store.Range(prefix)
}

func (s *StoreState) Get(key []byte) []byte {
	return s.store.Get(key)
}

func (s *StoreState) Has(key []byte) bool {
	return s.store.Has(key)
}

func (s *StoreState) Validators() []*loom.Validator {
	return s.validators.Slice()
}

func (s *StoreState) SetValidatorPower(pubKey []byte, power int64) {
	s.validators.Set(&types.Validator{PubKey: pubKey, Power: power})
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
		store:      s.store,
		block:      s.block,
		ctx:        ctx,
		validators: s.validators,
	}
}

func StateWithPrefix(prefix []byte, state State) State {
	return &StoreState{
		store:      store.PrefixKVStore(prefix, state),
		block:      state.Block(),
		ctx:        state.Context(),
		validators: loom.NewValidatorSet(state.Validators()...),
	}
}

type TxHandler interface {
	ProcessTx(state State, txBytes []byte) (TxHandlerResult, error)
}

type TxHandlerFunc func(state State, txBytes []byte) (TxHandlerResult, error)

type TxHandlerResult struct {
	Data             []byte
	ValidatorUpdates []abci.Validator
	Info             string
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tendermint/libs/pubsub/query)
	Tags             []common.KVPair
	Origin           loom.Address
}

func (f TxHandlerFunc) ProcessTx(state State, txBytes []byte) (TxHandlerResult, error) {
	return f(state, txBytes)
}

type QueryHandler interface {
	Handle(state ReadOnlyState, path string, data []byte) ([]byte, error)
}

type OrginHandler interface {
	ValidateDeployer(origin loom.Address) error
	ValidateCaller(origin loom.Address) error
	Clear(blockNumber int64)
}

type ValidatorsManager interface {
	BeginBlock(abci.RequestBeginBlock, string) error
	EndBlock(abci.RequestEndBlock) ([]abci.Validator, error)
}

type ValidatorsManagerFactoryFunc func(state State) (ValidatorsManager, error)

type Application struct {
	lastBlockHeader  abci.Header
	curBlockHeader   abci.Header
	curBlockHash     []byte
	validatorUpdates []types.Validator
	UseCheckTx       bool
	Store            store.VersionedKVStore
	Init             func(State) error
	TxHandler
	QueryHandler
	EventHandler
	ReceiptHandler         ReceiptHandler
	CreateValidatorManager ValidatorsManagerFactoryFunc
	OrginHandler
}

var _ abci.Application = &Application{}

//Metrics
var (
	deliverTxLatency    metrics.Histogram
	checkTxLatency      metrics.Histogram
	commitBlockLatency  metrics.Histogram
	requestCount        metrics.Counter
	committedBlockCount metrics.Counter
)

func init() {
	fieldKeys := []string{"method", "error"}
	requestCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	deliverTxLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "delivertx_latency_microseconds",
		Help:      "Total duration of delivertx in microseconds.",
	}, fieldKeys)

	checkTxLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "checktx_latency_microseconds",
		Help:      "Total duration of checktx in microseconds.",
	}, fieldKeys)
	commitBlockLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "commit_block_latency_microseconds",
		Help:      "Total duration of commit block in microseconds.",
	}, fieldKeys)

	committedBlockCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "block_count",
		Help:      "Number of committed blocks.",
	}, fieldKeys)
}

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
		nil,
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
	a.curBlockHash = req.Hash

	storeTx := store.WrapAtomic(a.Store).BeginTx()
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		nil,
	)

	validatorManager, err := a.CreateValidatorManager(state)
	if err != nil {
		panic(err)
	}

	err = validatorManager.BeginBlock(req, a.curBlockHeader.ChainID)
	if err != nil {
		panic(err)
	}

	storeTx.Commit()

	return abci.ResponseBeginBlock{}
}

func (a *Application) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	if req.Height != a.height() {
		panic("state version does not match end block height")
	}

	a.OrginHandler.Clear(a.curBlockHeader.Height)

	storeTx := store.WrapAtomic(a.Store).BeginTx()
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		nil,
	)

	if err := a.ReceiptHandler.CommitBlock(state, a.height()); err != nil {
		storeTx.Rollback()
		// TODO: maybe panic instead?
		log.Error(fmt.Sprintf("aborted committing block receipts, %v", err.Error()))
	} else {
		storeTx.Commit()
	}

	storeTx = store.WrapAtomic(a.Store).BeginTx()
	state = NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		nil,
	)

	validatorManager, err := a.CreateValidatorManager(state)
	if err != nil {
		panic(err)
	}
	validators, err := validatorManager.EndBlock(req)
	if err != nil {
		panic(err)
	}

	storeTx.Commit()

	return abci.ResponseEndBlock{
		ValidatorUpdates: validators,
	}
}

func (a *Application) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	ok := abci.ResponseCheckTx{Code: abci.CodeTypeOK}
	if !a.UseCheckTx {
		return ok
	}

	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "CheckTx", "error", fmt.Sprint(err != nil)}
		checkTxLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	// If the chain is configured not to generate empty blocks then CheckTx may be called before
	// BeginBlock when the application restarts, which means that both curBlockHeader and
	// lastBlockHeader will be default initialized. Instead of invoking a contract method with
	// a vastly innacurate block header simply skip invoking the contract. This has the minor
	// disadvantage of letting an potentially invalid tx propagate to other nodes, but this should
	// only happen on node restarts, and only if the node doesn't receive any txs from it's peers
	// before a client sends it a tx.
	if a.curBlockHeader.Height == 0 {
		return ok
	}

	_, err = a.processTx(txBytes, true)
	if err != nil {
		log.Error(fmt.Sprintf("CheckTx: %s", err.Error()))
		return abci.ResponseCheckTx{Code: 1, Log: err.Error()}
	}

	return ok
}
func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "DeliverTx", "error", fmt.Sprint(err != nil)}
		requestCount.With(lvs...).Add(1)
		deliverTxLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	r, err := a.processTx(txBytes, false)
	if err != nil {
		log.Error(fmt.Sprintf("DeliverTx: %s", err.Error()))
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Data: r.Data, Tags: r.Tags, Info: r.Info}
}

func (a *Application) processTx(txBytes []byte, fake bool) (TxHandlerResult, error) {
	var err error
	storeTx := store.WrapAtomic(a.Store).BeginTx()
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		a.curBlockHash,
	)

	r, err := a.TxHandler.ProcessTx(state, txBytes)
	if err != nil {
		storeTx.Rollback()
		// TODO: save receipt & hash of failed EVM tx to node-local persistent cache (not app state)
		a.ReceiptHandler.DiscardCurrentReceipt()
		return r, err
	}

	if fake {
		var err error
		if r.Info == utils.DeployEvm {
			err = a.OrginHandler.ValidateDeployer(r.Origin)
		} else 	if r.Info == utils.CallEVM {
			err = a.OrginHandler.ValidateCaller(r.Origin)
		}
		if err != nil {
			storeTx.Rollback()
			a.ReceiptHandler.DiscardCurrentReceipt()
			return r, err
		}
	}

	if !fake {
		if r.Info == utils.CallEVM || r.Info == utils.DeployEvm {
			a.EventHandler.EthSubscriptionSet().EmitTxEvent(r.Data, r.Info)
			a.ReceiptHandler.CommitCurrentReceipt()
		}
		storeTx.Commit()
	}
	return r, nil
}

// Commit commits the current block
func (a *Application) Commit() abci.ResponseCommit {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "Commit", "error", fmt.Sprint(err != nil)}
		committedBlockCount.With(lvs...).Add(1)
		commitBlockLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	appHash, _, err := a.Store.SaveVersion()
	if err != nil {
		panic(err)
	}
	height := a.curBlockHeader.GetHeight()
	a.EventHandler.EmitBlockTx(uint64(height))
	a.EventHandler.EthSubscriptionSet().EmitBlockEvent(a.curBlockHeader)
	a.lastBlockHeader = a.curBlockHeader

	if err := a.Store.Prune(); err != nil {
		log.Error("failed to prune app.db", "err", err)
	}

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
		nil,
	)
}
