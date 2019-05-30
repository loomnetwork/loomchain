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
	"github.com/loomnetwork/loomchain/db"
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
	// This is the prefix of versioning Patricia roots
	evmRootPrefix = []byte("evmroot")

	saveVersionDuration metrics.Histogram
	getSnapshotDuration metrics.Histogram
)

func init() {
	saveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: "loomchain",
			Subsystem: "multi_writer_appstore",
			Name:      "save_version",
			Help:      "How long MultiWriterAppStore.SaveVersion() took to execute (in seconds)",
		}, []string{},
	)

	getSnapshotDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: "loomchain",
			Subsystem: "multi_writer_appstore",
			Name:      "get_snapshot",
			Help:      "How long MultiWriterAppStore.GetSnapshot() took to execute (in seconds)",
		}, []string{},
	)
}

// MultiWriterAppStore reads & writes keys that have the "vm" prefix via both the IAVLStore and the EvmStore,
// or just the EvmStore, depending on the evmStoreEnabled flag.
type MultiWriterAppStore struct {
	appStore                   VersionedKVStore
	evmStore                   *EvmStore
	lastSavedTree              unsafe.Pointer // *iavl.ImmutableTree
	onlySaveEvmStateToEvmStore bool
	multiReaderIAVLStore       bool
}

// NewMultiWriterAppStore creates a new NewMultiWriterAppStore.
func NewMultiWriterAppStore(appStore VersionedKVStore, evmStore *EvmStore, saveEVMStateToIAVL, multiReaderIAVLStore bool) (*MultiWriterAppStore, error) {
	store := &MultiWriterAppStore{
		appStore:                   appStore,
		evmStore:                   evmStore,
		onlySaveEvmStateToEvmStore: !saveEVMStateToIAVL,
		multiReaderIAVLStore:       multiReaderIAVLStore,
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
	evmStoreEvmRoot, version := store.evmStore.getLastSavedRoot(store.appStore.Version())
	if !bytes.Equal(appStoreEvmRoot, evmStoreEvmRoot) {
		return nil, fmt.Errorf("EVM roots mismatch, evm.db(%d): %X, app.db(%d): %X",
			version, evmStoreEvmRoot, appStore.Version(), appStoreEvmRoot)
	}

	// feature flag overrides SaveEVMStateToIAVL
	if !store.onlySaveEvmStateToEvmStore {
		store.onlySaveEvmStateToEvmStore = bytes.Equal(store.appStore.Get(evmDBFeatureKey), []byte{1})
	}
	if !multiReaderIAVLStore {
		store.setLastSavedTreeToVersion(appStore.Version())
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
		s.appStore.Set(rootKey, currentRoot)
	}
	hash, version, err := s.appStore.SaveVersion()
	if !s.multiReaderIAVLStore {
		s.setLastSavedTreeToVersion(version)
	}

	return hash, version, err
}

func (s *MultiWriterAppStore) setLastSavedTreeToVersion(version int64) error {
	var err error
	var tree *iavl.ImmutableTree

	appStore := s.appStore.(*IAVLStore)
	if version == 0 {
		tree = iavl.NewImmutableTree(nil, 0)
	} else {
		tree, err = appStore.tree.GetImmutable(version)
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

	appStoreSnapshot := s.appStore.GetSnapshot()
	evmDbSnapshot := s.evmStore.GetSnapshot(s.appStore.Version())
	return newMultiWriterStoreSnapshot(evmDbSnapshot, appStoreSnapshot)
}

type multiWriterStoreSnapshot struct {
	evmDbSnapshot    db.Snapshot
	appStoreSnapshot Snapshot
}

func newMultiWriterStoreSnapshot(evmDbSnapshot db.Snapshot, appStoreSnapshot Snapshot) *multiWriterStoreSnapshot {
	return &multiWriterStoreSnapshot{
		evmDbSnapshot:    evmDbSnapshot,
		appStoreSnapshot: appStoreSnapshot,
	}
}

func (s *multiWriterStoreSnapshot) Release() {
	s.evmDbSnapshot.Release()
	s.appStoreSnapshot.Release()
}

func (s *multiWriterStoreSnapshot) Has(key []byte) bool {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmDbSnapshot.Has(key)
	}
	return s.appStoreSnapshot.Has(key)
}

func (s *multiWriterStoreSnapshot) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) {
		return s.evmDbSnapshot.Get(key)
	}
	return s.appStoreSnapshot.Get(key)
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
	return s.appStoreSnapshot.Range(prefix)
}
