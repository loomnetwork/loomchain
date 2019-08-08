package store

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/iavl"
)

var (
	// This is the same prefix as vmPrefix in evm/loomevm.go
	// We have to do this to avoid cyclic dependency
	vmPrefix = []byte("vm")
	// This is the same key as rootKey in evm/loomevm.go.
	rootKey = []byte("vmroot")
	// Using the same featurePrefix as in app.go, and the same EvmDBFeature name as in features.go
	evmDBFeatureKey = util.PrefixKey([]byte("feature"), []byte("db:evm"))
	// Using the same featurePrefix as in app.go, and the same AppStoreVersion3_1 name as in features.go
	appStoreVersion3_1 = util.PrefixKey([]byte("feature"), []byte("appstore:v3.1"))
	// This is the prefix of versioning Patricia roots
	evmRootPrefix = []byte("evmroot")

	saveVersionDuration metrics.Histogram
	getSnapshotDuration metrics.Histogram
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
			Help:       "How long MultiWriterAppStore.GetSnapshot() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)
}

// MultiWriterAppStore reads & writes keys that have the "vm" prefix via both the IAVLStore and the EvmStore,
// or just the EvmStore, depending on the evmStoreEnabled flag.
type MultiWriterAppStore struct {
	appStore                   *IAVLStore
	evmStore                   *EvmStore
	lastSavedTree              unsafe.Pointer // *iavl.ImmutableTree
	onlySaveEvmStateToEvmStore bool
}

// NewMultiWriterAppStore creates a new NewMultiWriterAppStore.
func NewMultiWriterAppStore(appStore *IAVLStore, evmStore *EvmStore, saveEVMStateToIAVL bool) (*MultiWriterAppStore, error) {
	store := &MultiWriterAppStore{
		appStore:                   appStore,
		evmStore:                   evmStore,
		onlySaveEvmStateToEvmStore: !saveEVMStateToIAVL,
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
	evmStoreEvmRoot, version := store.evmStore.GetLastSavedRoot(store.appStore.Version())
	if !bytes.Equal(appStoreEvmRoot, evmStoreEvmRoot) {
		return nil, fmt.Errorf("EVM roots mismatch, evm.db(%d): %X, app.db(%d): %X",
			version, evmStoreEvmRoot, appStore.Version(), appStoreEvmRoot)
	}

	// feature flag overrides SaveEVMStateToIAVL
	if !store.onlySaveEvmStateToEvmStore {
		store.onlySaveEvmStateToEvmStore = bytes.Equal(store.appStore.Get(evmDBFeatureKey), []byte{1})
	}

	store.setLastSavedTreeToVersion(appStore.Version())
	return store, nil
}

func LoadMultiWriterAppStore(appstoreCfg *AppStoreConfig, evmstoreCfg *EvmStoreConfig, appstoredb db.DBWrapper,
	targetVersion int64, metricsDatabase bool, rootPath string) (*MultiWriterAppStore, error) {
	iavlStore, err := NewIAVLStore(appstoredb, appstoreCfg.MaxVersions, targetVersion, appstoreCfg.IAVLFlushInterval)
	if err != nil {
		return nil, err
	}
	appStoreEvmRoot := iavlStore.Get(rootKey)
	evmDB, err := db.LoadDB(
		evmstoreCfg.DBBackend,
		evmstoreCfg.DBName,
		rootPath,
		evmstoreCfg.CacheSizeMegs,
		evmstoreCfg.WriteBufferMegs,
		metricsDatabase,
	)
	if err != nil {
		return nil, err
	}

	evmStore, err := loadEvmStore(evmDB, iavlStore.Version(), evmstoreCfg.NumCachedRoots)
	if err != nil {
		return nil, err
	}
	evmStoreEvmRoot, evmVersion := evmStore.GetLastSavedRoot(iavlStore.Version())
	if !bytes.Equal(appStoreEvmRoot, evmStoreEvmRoot) {
		log.Info(
			"EVM root mismatch, resyncing roots...",
			"evm.db-ver", evmVersion, "app.db-ver", iavlStore.Version(),
			"evm.db-root", hex.EncodeToString(evmStoreEvmRoot), "app.db-evm-root", hex.EncodeToString(appStoreEvmRoot),
		)
		latestVersion, err := iavl.NewMutableTree(appstoredb, 10000).LatestVersion(evmVersion)
		if err != nil {
			return nil, err
		}
		if latestVersion == 0 {
			latestVersion = -1
		}
		iavlStore, err = NewIAVLStore(appstoredb, appstoreCfg.MaxVersions, latestVersion, appstoreCfg.IAVLFlushInterval)
		if err != nil {
			return nil, err
		}
		evmStore.CloseDB()
		evmDB, err = db.LoadDB(
			evmstoreCfg.DBBackend,
			evmstoreCfg.DBName,
			rootPath,
			evmstoreCfg.CacheSizeMegs,
			evmstoreCfg.WriteBufferMegs,
			metricsDatabase,
		)
		if err != nil {
			return nil, err
		}
		evmStore, err = loadEvmStore(evmDB, iavlStore.Version(), evmstoreCfg.NumCachedRoots)
		if err != nil {
			return nil, err
		}
	}
	store := &MultiWriterAppStore{
		appStore:                   iavlStore,
		evmStore:                   evmStore,
		onlySaveEvmStateToEvmStore: !appstoreCfg.SaveEVMStateToIAVL,
	}
	// feature flag overrides SaveEVMStateToIAVL
	if !store.onlySaveEvmStateToEvmStore {
		store.onlySaveEvmStateToEvmStore = bytes.Equal(store.appStore.Get(evmDBFeatureKey), []byte{1})
	}
	err = store.setLastSavedTreeToVersion(iavlStore.Version())
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MultiWriterAppStore) Delete(key []byte) {
	if util.HasPrefix(key, vmPrefix) {
		s.evmStore.Delete(key)
		if !s.onlySaveEvmStateToEvmStore {
			s.appStore.Delete(key)
		}
	} else {
		s.appStore.Delete(key)
	}
}

func (s *MultiWriterAppStore) Set(key, val []byte) {
	if !s.onlySaveEvmStateToEvmStore && bytes.Equal(key, evmDBFeatureKey) {
		s.onlySaveEvmStateToEvmStore = bytes.Equal(val, []byte{1})
	}
	if util.HasPrefix(key, vmPrefix) {
		s.evmStore.Set(key, val)
		if !s.onlySaveEvmStateToEvmStore {
			s.appStore.Set(key, val)
		}
	} else {
		s.appStore.Set(key, val)
	}
}

func (s *MultiWriterAppStore) Has(key []byte) bool {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmStore.Has(key)
	}
	return s.appStore.Has(key)
}

func (s *MultiWriterAppStore) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmStore.Get(key)
	}
	return s.appStore.Get(key)
}

// Range iterates in-order over the keys in the store prefixed by the given prefix.
func (s *MultiWriterAppStore) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	if bytes.Equal(prefix, vmPrefix) || util.HasPrefix(prefix, vmPrefix) {
		return s.evmStore.Range(prefix)
	}
	return s.appStore.Range(prefix)
}

func (s *MultiWriterAppStore) Hash() []byte {
	return s.appStore.Hash()
}

func (s *MultiWriterAppStore) Version() int64 {
	return s.appStore.Version()
}

func (s *MultiWriterAppStore) SaveVersion() ([]byte, int64, error) {
	var err error
	defer func(begin time.Time) {
		saveVersionDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	currentRoot := s.evmStore.Commit(s.Version() + 1)
	if s.onlySaveEvmStateToEvmStore {
		// Tie up Patricia tree with IAVL tree.
		// Do this after the feature flag is enabled so that we can detect
		// inconsistency in evm.db across the cluster
		// AppStore 3.1 write EVM root to app.db only if it changes
		if bytes.Equal(s.appStore.Get(appStoreVersion3_1), []byte{1}) {
			oldRoot := s.appStore.Get(rootKey)
			if !bytes.Equal(oldRoot, currentRoot) {
				s.appStore.Set(rootKey, currentRoot)
			}
		} else {
			s.appStore.Set(rootKey, currentRoot)
		}

	}
	hash, version, err := s.appStore.SaveVersion()
	s.setLastSavedTreeToVersion(version)
	return hash, version, err
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

func (s *MultiWriterAppStore) GetSnapshot() Snapshot {
	defer func(begin time.Time) {
		getSnapshotDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())
	appStoreTree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
	evmDbSnapshot := s.evmStore.GetSnapshot(appStoreTree.Version())
	return newMultiWriterStoreSnapshot(evmDbSnapshot, appStoreTree)
}

type multiWriterStoreSnapshot struct {
	evmDbSnapshot db.Snapshot
	appStoreTree  *iavl.ImmutableTree
}

func newMultiWriterStoreSnapshot(evmDbSnapshot db.Snapshot, appStoreTree *iavl.ImmutableTree) *multiWriterStoreSnapshot {
	return &multiWriterStoreSnapshot{
		evmDbSnapshot: evmDbSnapshot,
		appStoreTree:  appStoreTree,
	}
}

func (s *multiWriterStoreSnapshot) Release() {
	s.evmDbSnapshot.Release()
	s.appStoreTree = nil
}

func (s *multiWriterStoreSnapshot) Has(key []byte) bool {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmDbSnapshot.Has(key)
	}
	return s.appStoreTree.Has(key)
}

func (s *multiWriterStoreSnapshot) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmDbSnapshot.Get(key)
	}
	_, val := s.appStoreTree.Get(key)
	return val
}

// Range iterates in-order over the keys in the store prefixed by the given prefix.
func (s *multiWriterStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	ret := make(plugin.RangeData, 0)

	if bytes.Equal(prefix, vmPrefix) || util.HasPrefix(prefix, vmPrefix) {
		it := s.evmDbSnapshot.NewIterator(prefix, prefixRangeEnd(prefix))
		defer it.Close()

		for ; it.Valid(); it.Next() {
			key := it.Key()
			if util.HasPrefix(key, prefix) {
				var err error
				key, err = util.UnprefixKey(key, prefix)
				if err != nil {
					panic(err)
				}

				ret = append(ret, &plugin.RangeEntry{
					Key:   key,
					Value: it.Value(),
				})
			}
		}
		return ret
	}

	// Otherwise iterate over the IAVL tree
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
