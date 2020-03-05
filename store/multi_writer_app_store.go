package store

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/iavl"
)

var (
	// This is the same prefix as vmPrefix in evm/loomevm.go
	// We have to do this to avoid cyclic dependency
	vmPrefix = []byte("vm")
	// This is the same key as rootKey in evm/loomevm.go
	rootKey = []byte("vmroot")

	saveVersionDuration  metrics.Histogram
	getSnapshotDuration  metrics.Histogram
	pruneEVMKeysDuration metrics.Histogram
	pruneEVMKeysCount    metrics.Counter
)

func init() {
	saveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  "loomchain",
			Subsystem:  "multi_writer_appstore",
			Name:       "save_version",
			Help:       "How long MultiWriterAppStore.SaveVersion() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)

	getSnapshotDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  "loomchain",
			Subsystem:  "multi_writer_appstore",
			Name:       "get_snapshot",
			Help:       "How long MultiWriterAppStore.GetSnapshotAt() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)

	pruneEVMKeysDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  "loomchain",
			Subsystem:  "multi_writer_appstore",
			Name:       "prune_evm_keys",
			Help:       "How long purning EVM keys took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)

	pruneEVMKeysCount = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: "loomchain",
			Subsystem: "multi_writer_appstore",
			Name:      "num_pruned_evm_keys",
			Help:      "Number of pruned EVM keys",
		}, []string{},
	)
}

// GetEVMRootFromAppStore retrieves the current EVM root from the given app store.
func GetEVMRootFromAppStore(s KVReader) []byte {
	evmRoot := s.Get(rootKey)
	if evmRoot == nil {
		return defaultRoot
	}
	return evmRoot
}

// MultiWriterAppStore keeps the EVM Patricia tree and IAVL tree roots in sync for each version so
// that all on-chain state can be consistently persisted & loaded at any height.
//
// A previous version of this store used to handle EVM state keys (denoted by the "vm" prefix) but
// the current version is only capable of pruning old EVM state keys from the IAVLStore, the EVM
// state keys are now handled by the EVMState.
type MultiWriterAppStore struct {
	appStore      *IAVLStore
	evmStore      *EvmStore
	lastSavedTree unsafe.Pointer // *iavl.ImmutableTree
}

// NewMultiWriterAppStore creates a new MultiWriterAppStore.
func NewMultiWriterAppStore(appStore *IAVLStore, evmStore *EvmStore) (*MultiWriterAppStore, error) {
	store := &MultiWriterAppStore{
		appStore: appStore,
		evmStore: evmStore,
	}
	appStoreEvmRoot := store.appStore.Get(rootKey)
	// if root is nil, this is the first run after migration, so get evmroot from vmvmroot
	if appStoreEvmRoot == nil {
		appStoreEvmRoot = store.appStore.Get(util.PrefixKey(vmPrefix, rootKey))
		// if root is still nil, evm state is empty, set appStoreEvmRoot to default root
		if appStoreEvmRoot == nil && store.appStore.Version() > 0 {
			appStoreEvmRoot = defaultRoot
		}
	}
	evmStoreEvmRoot, version := store.evmStore.GetRootAt(store.appStore.Version())
	if !bytes.Equal(appStoreEvmRoot, evmStoreEvmRoot) {
		return nil, fmt.Errorf("EVM roots mismatch, evm.db(%d): %X, app.db(%d): %X",
			version, evmStoreEvmRoot, appStore.Version(), appStoreEvmRoot)
	}

	if err := store.setLastSavedTreeToVersion(appStore.Version()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MultiWriterAppStore) Delete(key []byte) {
	s.appStore.Delete(key)
}

func (s *MultiWriterAppStore) Set(key, val []byte) {
	s.appStore.Set(key, val)
}

func (s *MultiWriterAppStore) Has(key []byte) bool {
	return s.appStore.Has(key)
}

func (s *MultiWriterAppStore) Get(key []byte) []byte {
	return s.appStore.Get(key)
}

// Range iterates in-order over the keys in the store prefixed by the given prefix.
func (s *MultiWriterAppStore) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	return s.appStore.Range(prefix)
}

func (s *MultiWriterAppStore) Hash() []byte {
	return s.appStore.Hash()
}

func (s *MultiWriterAppStore) Version() int64 {
	return s.appStore.Version()
}

func (s *MultiWriterAppStore) SaveVersion(opts *VersionedKVStoreSaveOptions) ([]byte, int64, error) {
	var err error
	defer func(begin time.Time) {
		saveVersionDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	var flushInterval int64
	if opts != nil {
		flushInterval = opts.FlushInterval
	}
	currentRoot := s.evmStore.Commit(s.Version()+1, flushInterval)
	// Store the root of the EVM Patricia tree in the IAVL tree.
	// Only write the EVM root to the IAVL store if it changes, this was previously only done
	// once the AppStoreVersion3_1 feature flag was enabled, but it's now assumed the flag is
	// always enabled so the feature check is omitted.
	oldRoot := s.appStore.Get(rootKey)
	if !bytes.Equal(oldRoot, currentRoot) {
		s.appStore.Set(rootKey, currentRoot)
	}

	if err := s.pruneOldEVMKeys(); err != nil {
		return nil, 0, err
	}

	hash, version, err := s.appStore.SaveVersion(opts)
	s.setLastSavedTreeToVersion(version)
	return hash, version, err
}

func (s *MultiWriterAppStore) pruneOldEVMKeys() error {
	defer func(begin time.Time) {
		pruneEVMKeysDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	// TODO: Rather than loading the on-chain config here relevant setting should be passed in as a
	//       parameter to SaveVersion().
	cfg, err := LoadOnChainConfig(s.appStore)
	if err != nil {
		return err
	}

	maxKeysToPrune := cfg.GetAppStore().GetNumEvmKeysToPrune()
	pruneInterval := cfg.GetAppStore().GetPruneEvmKeysInterval()
	// If pruneInterval is set, then only prune old EVM Keys every N blocks
	if (pruneInterval != 0) && (s.Version()%int64(pruneInterval) != 0) {
		maxKeysToPrune = 0
	}
	if maxKeysToPrune > 0 {
		entriesToPrune := s.appStore.RangeWithLimit(vmPrefix, int(maxKeysToPrune))
		for _, entry := range entriesToPrune {
			s.appStore.Delete(util.PrefixKey(vmPrefix, entry.Key))
		}
		pruneEVMKeysCount.Add(float64(len(entriesToPrune)))
	}
	return nil
}

func (s *MultiWriterAppStore) setLastSavedTreeToVersion(version int64) error {
	var err error
	var tree *iavl.ImmutableTree

	if version == 0 {
		tree = iavl.NewImmutableTree(nil, 0)
	} else {
		tree, err = s.appStore.tree.GetImmutable(version)
		if err != nil {
			return errors.Wrapf(err, "failed to load immutable tree for version %v", version)
		}
	}

	atomic.StorePointer(&s.lastSavedTree, unsafe.Pointer(tree))
	return nil
}

func (s *MultiWriterAppStore) Prune() error {
	return s.appStore.Prune()
}

func (s *MultiWriterAppStore) GetSnapshotAt(version int64) (Snapshot, error) {
	defer func(begin time.Time) {
		getSnapshotDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	var err error
	var appStoreTree *iavl.ImmutableTree
	previousTree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.appStore.previousTree))
	if version == 0 {
		appStoreTree = (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
	} else if previousTree != nil && previousTree.Version() == version {
		appStoreTree = previousTree
	} else {
		appStoreTree, err = s.appStore.tree.GetImmutable(version)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load immutable tree for version %v", version)
		}
	}
	return newMultiWriterStoreSnapshot(appStoreTree), nil
}

type multiWriterStoreSnapshot struct {
	appStoreTree *iavl.ImmutableTree
}

func newMultiWriterStoreSnapshot(appStoreTree *iavl.ImmutableTree) *multiWriterStoreSnapshot {
	return &multiWriterStoreSnapshot{
		appStoreTree: appStoreTree,
	}
}

func (s *multiWriterStoreSnapshot) Release() {
	s.appStoreTree = nil
}

func (s *multiWriterStoreSnapshot) Has(key []byte) bool {
	return s.appStoreTree.Has(key)
}

func (s *multiWriterStoreSnapshot) Get(key []byte) []byte {
	_, val := s.appStoreTree.Get(key)
	return val
}

// Range iterates in-order over the keys in the store prefixed by the given prefix.
func (s *multiWriterStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	ret := make(plugin.RangeData, 0)
	keys, values, _, err := s.appStoreTree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
	if err != nil {
		log.Error("failed to get range", "prefix", string(prefix), "err", err)
		return ret
	}

	for i, k := range keys {
		// Tree range gives all keys that has prefix but it does not check zero byte
		// after the prefix. So we have to check zero byte after prefix using util.HasPrefix
		if util.HasPrefix(k, prefix) {
			k, err = util.UnprefixKey(k, prefix)
			if err != nil {
				panic(err)
			}
		} else { // Skip this key as it does not have the prefix
			continue
		}

		ret = append(ret, &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		})
	}

	return ret
}
