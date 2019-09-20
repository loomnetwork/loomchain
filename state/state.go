package state

import (
	"bytes"
	"context"

	"github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/config"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/store"
)

type GetValidatorSet func(state State) (loom.ValidatorSet, error)
type ReadOnlyState interface {
	store.KVReader
	Validators() []*loom.Validator
	Block() types.BlockHeader
	// Release should free up any underlying system resources. Must be safe to invoke multiple times.
	Release()
	FeatureEnabled(string, bool) bool
	Config() *cctypes.Config
	EnabledFeatures() []string
}

type State interface {
	ReadOnlyState
	store.KVWriter
	Context() context.Context
	WithContext(ctx context.Context) State
	WithPrefix(prefix []byte) State
	SetFeature(string, bool)
	ChangeConfigSetting(name, value string) error
}

type StoreState struct {
	ctx             context.Context
	store           store.KVStore
	block           types.BlockHeader
	validators      loom.ValidatorSet
	getValidatorSet GetValidatorSet
	config          *cctypes.Config
}

var _ = State(&StoreState{})

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

const (
	featurePrefix = "feature"
)

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

// ChangeConfigSetting updates the value of the given on-chain config setting.
// If an error occurs while trying to update the config the change is rolled back, if the rollback
// itself fails this function will panic.
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
	}
}

func (s *StoreState) WithPrefix(prefix []byte) State {
	return &StoreState{
		store:           store.PrefixKVStore(prefix, s.store),
		block:           s.block,
		ctx:             s.ctx,
		validators:      s.validators,
		getValidatorSet: s.getValidatorSet,
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

func featureKey(featureName string) []byte {
	return util.PrefixKey([]byte(featurePrefix), []byte(featureName))
}

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
