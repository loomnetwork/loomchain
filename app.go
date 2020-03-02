package loomchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/loomnetwork/go-loom/config"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/registry"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	blockindex "github.com/loomnetwork/loomchain/store/block_index"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
	ttypes "github.com/tendermint/tendermint/types"
)

type ReadOnlyState interface {
	store.KVReader
	Validators() []*loom.Validator
	Block() types.BlockHeader
	// Release should free up any underlying system resources. Must be safe to invoke multiple times.
	Release()
	FeatureEnabled(string, bool) bool
	Config() *cctypes.Config
	EnabledFeatures() []string
	GetMinBuildNumber() uint64
}

type State interface {
	ReadOnlyState
	store.KVWriter
	Context() context.Context
	WithContext(ctx context.Context) State
	WithPrefix(prefix []byte) State
	SetFeature(string, bool)
	SetMinBuildNumber(uint64)
	ChangeConfigSetting(name, value string) error
	EVMState() *EVMState
}

type StoreState struct {
	ctx             context.Context
	store           store.KVStore
	block           types.BlockHeader
	validators      loom.ValidatorSet
	getValidatorSet GetValidatorSet
	config          *cctypes.Config
	evmState        *EVMState
}

var _ = State(&StoreState{})

func blockHeaderFromAbciHeader(header *abci.Header) types.BlockHeader {
	return types.BlockHeader{
		ChainID: header.ChainID,
		Height:  header.Height,
		Time:    header.Time.Unix(),
		NumTxs:  int32(header.NumTxs), //TODO this cast doesnt look right
		LastBlockID: types.BlockID{
			Hash: header.LastBlockId.Hash,
		},
		ValidatorsHash: header.ValidatorsHash,
		AppHash:        header.AppHash,
	}
}

func NewStoreState(
	ctx context.Context,
	store store.KVStore,
	block abci.Header,
	curBlockHash []byte,
	getValidatorSet GetValidatorSet,
) *StoreState {
	blockHeader := blockHeaderFromAbciHeader(&block)
	blockHeader.CurrentHash = curBlockHash
	return &StoreState{
		ctx:             ctx,
		store:           store,
		block:           blockHeader,
		validators:      loom.NewValidatorSet(),
		getValidatorSet: getValidatorSet,
	}
}

func (s *StoreState) WithOnChainConfig(config *cctypes.Config) *StoreState {
	s.config = config
	return s
}

func (s *StoreState) WithEVMState(evmState *EVMState) *StoreState {
	s.evmState = evmState
	return s
}

func (s *StoreState) Range(prefix []byte) plugin.RangeData {
	return s.store.Range(prefix)
}

func (s *StoreState) Get(key []byte) []byte {
	return s.store.Get(key)
}

func (s *StoreState) Has(key []byte) bool {
	return s.store.Has(key)
}

func (s *StoreState) Validators() []*loom.Validator {
	if (len(s.validators) == 0) && (s.getValidatorSet != nil) {
		validatorSet, err := s.getValidatorSet(s)
		if err != nil {
			panic(err)
		}
		// cache the validator set for the current state
		s.validators = validatorSet
	}
	return s.validators.Slice()
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

func (s *StoreState) EVMState() *EVMState {
	return s.evmState
}

const (
	featurePrefix = "feature"
	MinBuildKey   = "minbuild"
)

func featureKey(featureName string) []byte {
	return util.PrefixKey([]byte(featurePrefix), []byte(featureName))
}

func (s *StoreState) EnabledFeatures() []string {
	featuresFromState := s.Range([]byte(featurePrefix))
	enabledFeatures := make([]string, 0, len(featuresFromState))
	for _, m := range featuresFromState {
		if bytes.Equal(m.Value, []byte{1}) {
			enabledFeatures = append(enabledFeatures, string(m.Key))
		}
	}

	return enabledFeatures
}

func (s *StoreState) FeatureEnabled(name string, val bool) bool {
	data := s.store.Get(featureKey(name))
	if len(data) == 0 {
		return val
	}
	if bytes.Equal(data, []byte{1}) {
		return true
	}
	return false
}

func (s *StoreState) SetFeature(name string, val bool) {
	data := []byte{0}
	if val {
		data = []byte{1}
	}
	s.store.Set(featureKey(name), data)
}

// SetMinBuildNumber sets the minimum build number all nodes must be running.
func (s *StoreState) SetMinBuildNumber(minBuild uint64) {
	buildBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(buildBytes, minBuild)
	s.store.Set([]byte(MinBuildKey), buildBytes)
}

// GetMinBuildNumber returns the minimum build number all nodes must be running.
func (s *StoreState) GetMinBuildNumber() uint64 {
	buildBytes := s.store.Get([]byte(MinBuildKey))
	if len(buildBytes) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(buildBytes)
}

// ChangeConfigSetting updates the value of the given on-chain config setting.
// If an error occurs while trying to update the config the change is discarded.
func (s *StoreState) ChangeConfigSetting(name, value string) error {
	cfg, err := store.LoadOnChainConfig(s.store)
	if err != nil {
		panic(err)
	}
	if err := config.SetConfigSetting(cfg, name, value); err != nil {
		return err
	}
	if err := store.SaveOnChainConfig(s.store, cfg); err != nil {
		return err
	}
	// invalidate cached config so it's reloaded next time it's accessed
	s.config = nil
	return nil
}

// Config returns the current on-chain config.
func (s *StoreState) Config() *cctypes.Config {
	if s.config == nil {
		var err error
		s.config, err = store.LoadOnChainConfig(s.store)
		if err != nil {
			panic(err)
		}
	}
	return s.config
}

func (s *StoreState) WithContext(ctx context.Context) State {
	return &StoreState{
		store:           s.store,
		block:           s.block,
		ctx:             ctx,
		validators:      s.validators,
		getValidatorSet: s.getValidatorSet,
		evmState:        s.evmState,
	}
}

func (s *StoreState) WithPrefix(prefix []byte) State {
	return &StoreState{
		store:           store.PrefixKVStore(prefix, s.store),
		block:           s.block,
		ctx:             s.ctx,
		validators:      s.validators,
		getValidatorSet: s.getValidatorSet,
		evmState:        s.evmState,
	}
}

func (s *StoreState) Release() {
	// noop
}

// StoreStateSnapshot is a read-only snapshot of the app state at particular point in time,
// it's unaffected by any changes to the app state. Multiple snapshots can exist at any one
// time, but each snapshot should only be accessed from one thread at a time. After a snapshot
// is no longer needed call Release() to free up underlying resources. Live snapshots may prevent
// the underlying DB from writing new data in the most space efficient manner, so aim to minimize
// their lifetime.
type StoreStateSnapshot struct {
	*StoreState
	storeSnapshot store.Snapshot
}

// TODO: Ideally StoreStateSnapshot should only implement ReadOnlyState interface, but that will
//       require updating a bunch of the existing State consumers to also handle ReadOnlyState.
var _ = State(&StoreStateSnapshot{})

// NewStoreStateSnapshot creates a new snapshot of the app state.
func NewStoreStateSnapshot(
	ctx context.Context, snap store.Snapshot, block abci.Header, curBlockHash []byte,
	getValidatorSet GetValidatorSet,
) *StoreStateSnapshot {
	return &StoreStateSnapshot{
		StoreState:    NewStoreState(ctx, &readOnlyKVStoreAdapter{snap}, block, curBlockHash, getValidatorSet),
		storeSnapshot: snap,
	}
}

// Release releases the underlying store snapshot, safe to call multiple times.
func (s *StoreStateSnapshot) Release() {
	if s.storeSnapshot != nil {
		s.storeSnapshot.Release()
		s.storeSnapshot = nil
	}
}

// For all the times you need a read-only store.KVStore but you only have a store.KVReader.
type readOnlyKVStoreAdapter struct {
	store.KVReader
}

func (s *readOnlyKVStoreAdapter) Set(key, value []byte) {
	panic("kvStoreSnapshotAdapter.Set not implemented")
}

func (s *readOnlyKVStoreAdapter) Delete(key []byte) {
	panic("kvStoreSnapshotAdapter.Delete not implemented")
}

type TxHandler interface {
	ProcessTx(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error)
}

type TxHandlerFunc func(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error)

type TxHandlerResult struct {
	Data             []byte
	ValidatorUpdates []abci.Validator
	Info             string
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tendermint/libs/pubsub/query)
	Tags []common.KVPair
}

func (f TxHandlerFunc) ProcessTx(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	return f(state, txBytes, isCheckTx)
}

type QueryHandler interface {
	Handle(state ReadOnlyState, path string, data []byte) ([]byte, error)
}

type KarmaHandler interface {
	Upkeep() error
}

type ValidatorsManager interface {
	BeginBlock(abci.RequestBeginBlock, int64) error
	EndBlock(abci.RequestEndBlock) ([]abci.ValidatorUpdate, error)
}

type ChainConfigManager interface {
	EnableFeatures(blockHeight int64) error
	UpdateConfig() (int, error)
}

type GetValidatorSet func(state State) (loom.ValidatorSet, error)

type ValidatorsManagerFactoryFunc func(state State) (ValidatorsManager, error)

type ChainConfigManagerFactoryFunc func(state State) (ChainConfigManager, error)

type CommittedTx struct {
	result TxHandlerResult
	txHash []byte
}

type ApplicationParams struct {
	Store store.VersionedKVStore
	Init  func(State) error
	TxHandler
	QueryHandler
	EventHandler
	ReceiptHandlerProvider
	EvmAuxStore *evmaux.EvmAuxStore
	blockindex.BlockIndexStore
	CreateValidatorManager   ValidatorsManagerFactoryFunc
	CreateChainConfigManager ChainConfigManagerFactoryFunc
	// Callback function used to construct a contract upkeep handler at the start of each block,
	// should return a nil handler when the contract upkeep feature is disabled.
	CreateContractUpkeepHandler func(state State) (KarmaHandler, error)
	GetValidatorSet             GetValidatorSet
	EventStore                  store.EventStore
	ReceiptsVersion             int32
	EVMState                    *EVMState
}

type Application struct {
	ApplicationParams
	lastBlockHeader unsafe.Pointer // *abci.Header
	curBlockHeader  abci.Header
	curBlockHash    []byte
	config          *cctypes.Config
	childTxRefs     []evmaux.ChildTxRef // links Tendermint txs to EVM txs
	committedTxs    []CommittedTx
}

var _ abci.Application = &Application{}

//Metrics
var (
	deliverTxLatency     metrics.Histogram
	checkTxLatency       metrics.Histogram
	commitBlockLatency   metrics.Histogram
	beginBlockLatency    metrics.Histogram
	endBlockLatency      metrics.Histogram
	requestCount         metrics.Counter
	committedBlockCount  metrics.Counter
	validatorFuncLatency metrics.Histogram
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
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "delivertx_latency_microseconds",
		Help:       "Total duration of delivertx in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"method", "error", "evm"})

	checkTxLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "checktx_latency_microseconds",
		Help:       "Total duration of checktx in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
	commitBlockLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "commit_block_latency_microseconds",
		Help:       "Total duration of commit block in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
	beginBlockLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "begin_block_latency",
		Help:       "Total duration of begin block in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"method"})
	endBlockLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "end_block_latency",
		Help:       "Total duration of end block in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"method"})

	committedBlockCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "block_count",
		Help:      "Number of committed blocks.",
	}, fieldKeys)

	validatorFuncLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "validator_election_latency",
		Help:       "Total duration of validator election in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{})
}

func NewApplication(params ApplicationParams, lastBlockHeader *abci.Header) *Application {
	a := &Application{ApplicationParams: params}
	if lastBlockHeader != nil {
		atomic.StorePointer(&a.lastBlockHeader, unsafe.Pointer(&lastBlockHeader))
	}
	return a
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
		abci.Header{
			ChainID: req.ChainId,
		},
		nil,
		a.GetValidatorSet,
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
	defer func(begin time.Time) {
		lvs := []string{"method", "BeginBlock"}
		beginBlockLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	block := req.Header
	if block.Height != a.height() {
		panic(fmt.Sprintf("app height %d doesn't match BeginBlock height %d", a.height(), block.Height))
	}

	if a.config == nil {
		var err error
		a.config, err = store.LoadOnChainConfig(a.Store)
		if err != nil {
			panic(err)
		}
	}

	a.curBlockHeader = block
	a.curBlockHash = req.Hash

	if a.CreateContractUpkeepHandler != nil {
		upkeepStoreTx := store.WrapAtomic(a.Store).BeginTx()
		upkeepState := NewStoreState(
			context.Background(),
			upkeepStoreTx,
			a.curBlockHeader,
			a.curBlockHash,
			a.GetValidatorSet,
		).WithOnChainConfig(a.config).WithEVMState(a.EVMState)
		contractUpkeepHandler, err := a.CreateContractUpkeepHandler(upkeepState)
		if err != nil {
			panic(err)
		}
		if contractUpkeepHandler != nil {
			if err := contractUpkeepHandler.Upkeep(); err != nil {
				panic(err)
			}
			upkeepStoreTx.Commit()
		}
	}

	storeTx := store.WrapAtomic(a.Store).BeginTx()
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		nil,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config).WithEVMState(a.EVMState)

	validatorManager, err := a.CreateValidatorManager(state)
	if err != registry.ErrNotFound {
		if err != nil {
			panic(err)
		}

		err = validatorManager.BeginBlock(req, a.height())
		if err != nil {
			panic(err)
		}
	}

	//Enable Features
	chainConfigManager, err := a.CreateChainConfigManager(state)
	if err != nil {
		panic(err)
	}
	if chainConfigManager != nil {
		if err := chainConfigManager.EnableFeatures(a.height()); err != nil {
			panic(err)
		}

		numConfigChanges, err := chainConfigManager.UpdateConfig()
		if err != nil {
			panic(err)
		}

		if numConfigChanges > 0 {
			// invalidate cached config so it's reloaded next time it's accessed
			a.config = nil
		}
	}

	storeTx.Commit()

	return abci.ResponseBeginBlock{}
}

func (a *Application) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	defer func(begin time.Time) {
		lvs := []string{"method", "EndBlock"}
		endBlockLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	if req.Height != a.height() {
		panic(fmt.Sprintf("app height %d doesn't match EndBlock height %d", a.height(), req.Height))
	}

	// TODO: receiptHandler.CommitBlock() should be moved to Application.Commit()
	storeTx := store.WrapAtomic(a.Store).BeginTx()
	receiptHandler := a.ReceiptHandlerProvider.Store()
	if err := receiptHandler.CommitBlock(a.height()); err != nil {
		storeTx.Rollback()
		// TODO: maybe panic instead?
		log.Error(fmt.Sprintf("aborted committing block receipts, %v", err.Error()))
	} else {
		storeTx.Commit()
	}

	storeTx = store.WrapAtomic(a.Store).BeginTx()
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		nil,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config).WithEVMState(a.EVMState)

	validatorManager, err := a.CreateValidatorManager(state)
	if err != registry.ErrNotFound {
		if err != nil {
			panic(err)
		}
		t2 := time.Now()
		validators, err := validatorManager.EndBlock(req)

		diffsecs := time.Since(t2).Seconds()
		validatorFuncLatency.Observe(diffsecs)

		log.Info(fmt.Sprintf("validator manager took %f seconds-----\n", diffsecs))
		if err != nil {
			panic(err)
		}
		storeTx.Commit()

		return abci.ResponseEndBlock{
			ValidatorUpdates: validators,
		}
	}
	return abci.ResponseEndBlock{
		ValidatorUpdates: []abci.ValidatorUpdate{},
	}
}

func (a *Application) CheckTx(txBytes []byte) abci.ResponseCheckTx {
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
		return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
	}

	storeTx := store.WrapAtomic(a.Store).BeginTx()
	defer storeTx.Rollback()

	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		a.curBlockHash,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config)

	if a.EVMState != nil {
		state = state.WithEVMState(a.EVMState.Clone())
	}

	// Receipts & events generated in CheckTx must be discarded since the app state changes they
	// reflect aren't persisted.
	defer a.ReceiptHandlerProvider.Store().DiscardCurrentReceipt()
	defer a.EventHandler.Rollback()

	_, err = a.TxHandler.ProcessTx(state, txBytes, true)
	if err != nil {
		log.Error("CheckTx", "tx", hex.EncodeToString(ttypes.Tx(txBytes).Hash()), "err", err)
		return abci.ResponseCheckTx{Code: 1, Log: err.Error()}
	}

	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	var txFailed, isEvmTx bool
	defer func(begin time.Time) {
		lvs := []string{
			"method", "DeliverTx",
			"error", fmt.Sprint(txFailed),
			"evm", fmt.Sprint(isEvmTx),
		}
		requestCount.With(lvs[:4]...).Add(1)
		deliverTxLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	storeTx := store.WrapAtomic(a.Store).BeginTx()
	defer storeTx.Rollback()

	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		a.curBlockHash,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config).WithEVMState(a.EVMState)

	var r abci.ResponseDeliverTx

	if state.FeatureEnabled(features.EvmTxReceiptsVersion3_1, false) {
		r = a.deliverTx2(storeTx, txBytes)
	} else {
		r = a.deliverTx(storeTx, txBytes)
	}

	txFailed = r.Code != abci.CodeTypeOK
	// TODO: this isn't 100% reliable when txFailed == true
	isEvmTx = r.Info == utils.CallEVM || r.Info == utils.DeployEvm
	return r
}

// This version of DeliverTx doesn't store the receipts for failed EVM txs.
func (a *Application) deliverTx(storeTx store.KVStoreTx, txBytes []byte) abci.ResponseDeliverTx {
	r, err := a.processTx(storeTx, txBytes, false)
	if err != nil {
		log.Error("DeliverTx", "tx", hex.EncodeToString(ttypes.Tx(txBytes).Hash()), "err", err)
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Data: r.Data, Tags: r.Tags, Info: r.Info}
}

func (a *Application) processTx(storeTx store.KVStoreTx, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		a.curBlockHash,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config).WithEVMState(a.EVMState)

	receiptHandler := a.ReceiptHandlerProvider.Store()
	defer receiptHandler.DiscardCurrentReceipt()
	defer a.EventHandler.Rollback()

	r, err := a.TxHandler.ProcessTx(state, txBytes, isCheckTx)
	if err != nil {
		return r, err
	}

	if !isCheckTx {
		a.EventHandler.Commit(uint64(a.curBlockHeader.GetHeight()))

		saveEvmTxReceipt := r.Info == utils.CallEVM || r.Info == utils.DeployEvm ||
			state.FeatureEnabled(features.EvmTxReceiptsVersion3, false) || a.ReceiptsVersion == 3

		if saveEvmTxReceipt {
			if err := a.EventHandler.LegacyEthSubscriptionSet().EmitTxEvent(r.Data, r.Info); err != nil {
				log.Error("Emit Tx Event error", "err", err)
			}

			reader := a.ReceiptHandlerProvider.Reader()
			if reader.GetCurrentReceipt() != nil {
				receiptTxHash := reader.GetCurrentReceipt().TxHash
				if err := a.EventHandler.EthSubscriptionSet().EmitTxEvent(receiptTxHash); err != nil {
					log.Error("failed to emit tx event to subscribers", "err", err)
				}
				txHash := ttypes.Tx(txBytes).Hash()
				// If a receipt was generated for an EVM tx add a link between the TM tx hash and the EVM tx hash
				// so that we can use it to lookup relevant events using the TM tx hash.
				if !bytes.Equal(txHash, receiptTxHash) {
					a.childTxRefs = append(a.childTxRefs, evmaux.ChildTxRef{
						ParentTxHash: txHash,
						ChildTxHash:  receiptTxHash,
					})
				}
				receiptHandler.CommitCurrentReceipt()
			}
		}
		storeTx.Commit()
	}
	return r, nil
}

// This version of DeliverTx stores the receipts for failed EVM txs.
func (a *Application) deliverTx2(storeTx store.KVStoreTx, txBytes []byte) abci.ResponseDeliverTx {
	state := NewStoreState(
		context.Background(),
		storeTx,
		a.curBlockHeader,
		a.curBlockHash,
		a.GetValidatorSet,
	).WithOnChainConfig(a.config).WithEVMState(a.EVMState)

	receiptHandler := a.ReceiptHandlerProvider.Store()
	defer receiptHandler.DiscardCurrentReceipt()
	defer a.EventHandler.Rollback()

	r, txErr := a.TxHandler.ProcessTx(state, txBytes, false)

	// Store the receipt even if the tx itself failed
	var receiptTxHash []byte
	if a.ReceiptHandlerProvider.Reader().GetCurrentReceipt() != nil {
		receiptTxHash = a.ReceiptHandlerProvider.Reader().GetCurrentReceipt().TxHash
		txHash := ttypes.Tx(txBytes).Hash()
		// If a receipt was generated for an EVM tx add a link between the TM tx hash and the EVM tx hash
		// so that we can use it to lookup relevant events using the TM tx hash.
		if !bytes.Equal(txHash, receiptTxHash) {
			a.childTxRefs = append(a.childTxRefs, evmaux.ChildTxRef{
				ParentTxHash: txHash,
				ChildTxHash:  receiptTxHash,
			})
		}
		receiptHandler.CommitCurrentReceipt()
	}

	if txErr != nil {
		log.Error("DeliverTx", "tx", hex.EncodeToString(ttypes.Tx(txBytes).Hash()), "err", txErr)
		// FIXME: Really shouldn't be using r.Data if txErr != nil, but need to refactor TxHandler.ProcessTx
		//        so it only returns r with the correct status code & log fields.
		// Pass the EVM tx hash (if any) back to Tendermint so it stores it in block results
		return abci.ResponseDeliverTx{Code: 1, Data: r.Data, Log: txErr.Error()}
	}

	a.EventHandler.Commit(uint64(a.curBlockHeader.GetHeight()))
	storeTx.Commit()

	a.committedTxs = append(a.committedTxs, CommittedTx{
		result: r,
		txHash: receiptTxHash,
	})

	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK, Data: r.Data, Tags: r.Tags, Info: r.Info}
}

// Commit commits the current block
func (a *Application) Commit() abci.ResponseCommit {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "Commit", "error", fmt.Sprint(err != nil)}
		committedBlockCount.With(lvs...).Add(1)
		commitBlockLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	if a.EVMState != nil {
		// Commit EVM state changes to the EvmStore
		if err := a.EVMState.Commit(); err != nil {
			panic(err)
		}
	}

	storeOpts := store.VersionedKVStoreSaveOptions{
		FlushInterval: int64(a.config.GetAppStore().GetIAVLFlushInterval()),
	}

	appHash, _, err := a.Store.SaveVersion(&storeOpts)
	if err != nil {
		panic(err)
	}

	height := a.curBlockHeader.GetHeight()
	if err := a.EvmAuxStore.SaveChildTxRefs(a.childTxRefs); err != nil {
		// TODO: consider panic instead
		log.Error("Failed to save Tendermint -> EVM tx hash refs", "height", height, "err", err)
	}
	a.childTxRefs = nil

	// Update the index before emitting events in case the subscribers attempt to lookup the
	// block by number as soon as they receive an event.
	if a.BlockIndexStore != nil {
		a.BlockIndexStore.SetBlockHashAtHeight(uint64(height), a.curBlockHash)
	}

	// Update the last block header before emitting events in case the subscribers attempt to access
	// the latest committed state as soon as they receive an event.
	curBlockHeader := a.curBlockHeader
	atomic.StorePointer(&a.lastBlockHeader, unsafe.Pointer(&curBlockHeader))

	go func(height int64, blockHeader abci.Header, committedTxs []CommittedTx) {
		if err := a.EventHandler.EmitBlockTx(uint64(height), blockHeader.Time); err != nil {
			log.Error("Emit Block Event error", "err", err)
		}
		if err := a.EventHandler.LegacyEthSubscriptionSet().EmitBlockEvent(blockHeader); err != nil {
			log.Error("Emit Block Event error", "err", err)
		}
		if err := a.EventHandler.EthSubscriptionSet().EmitBlockEvent(blockHeader); err != nil {
			log.Error("Emit Block Event error", "err", err)
		}
		for _, tx := range committedTxs {
			if err := a.EventHandler.LegacyEthSubscriptionSet().EmitTxEvent(tx.result.Data, tx.result.Info); err != nil {
				log.Error("Emit Tx Event error", "err", err)
			}

			if len(tx.txHash) > 0 {
				if err := a.EventHandler.EthSubscriptionSet().EmitTxEvent(tx.txHash); err != nil {
					log.Error("failed to emit tx event to subscribers", "err", err)
				}
			}
		}
	}(height, a.curBlockHeader, a.committedTxs)
	a.committedTxs = nil

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
	lastBlockHeader := (*abci.Header)(atomic.LoadPointer(&a.lastBlockHeader))

	// When the node is started with no previous blockchain state (e.g. completely new chain) then
	// there'll be a very brief period where lastBlockHeader will be nil (until Application.Commit is called for the
	// first time). While lastBlockHeader is nil the node won't be able to return useful responses to most queries,
	// so we just make it panic here so the clients get an obvious error.
	// TODO: This is just quick hack, the proper way to deal with this scenario is to start the QueryServer only after
	// the lastBlockHeader has been set.
	if lastBlockHeader == nil {
		panic(errors.New("unable to respond to query, app isn't ready yet"))
	}

	appStateSnapshot, err := a.Store.GetSnapshotAt(lastBlockHeader.Height)
	if err != nil {
		panic(err)
	}

	var evmStateSnapshot *EVMState
	if a.EVMState != nil {
		evmStateSnapshot, err = a.EVMState.GetSnapshot(
			lastBlockHeader.Height,
			store.GetEVMRootFromAppStore(appStateSnapshot),
		)
		if err != nil {
			panic(err)
		}
	}

	return NewStoreStateSnapshot(
		nil,
		appStateSnapshot,
		*lastBlockHeader,
		nil, // TODO: last block hash!
		a.GetValidatorSet,
	).WithEVMState(evmStateSnapshot)
}
