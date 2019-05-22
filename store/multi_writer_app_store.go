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
	// This is the same key as featureKey in app.go
	featureKey = []byte("feature")
	// This is the same feature name as EvmDBFeature in features.go
	evmDBFeature = []byte("db:evm")
	// This is the prefix of versioning Patricia roots
	evmRootPrefix = []byte("evmroot")

	saveVersionDuration metrics.Histogram
)

func init() {
	const namespace = "loomchain"
	const subsystem = "multi_writer_appstore"

	saveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "save_version",
			Help:      "How long MultiWriterAppStore.SaveVersion() took to execute (in seconds)",
		}, []string{"error"})
}

// MultiWriterAppStore reads & writes keys that have the "vm" prefix via both the IAVLStore and the EvmStore,
// or just the EvmStore, depending on the evmStoreEnabled flag.
type MultiWriterAppStore struct {
	appStore        *IAVLStore
	evmStore        *EvmStore
	lastSavedTree   unsafe.Pointer // *iavl.ImmutableTree
	evmStoreEnabled bool
}

// NewMultiWriterAppStore creates a new NewMultiWriterAppStore.
func NewMultiWriterAppStore(appStore *IAVLStore, evmStore *EvmStore, evmStoreEnabled bool) (*MultiWriterAppStore, error) {
	store := &MultiWriterAppStore{
		evmStoreEnabled: evmStoreEnabled,
		appStore:        appStore,
		evmStore:        evmStore,
	}
	appStoreEvmRoot := store.appStore.Get(rootKey)
	evmStoreEvmRoot := store.evmStore.Get(rootHashKey)
	if !bytes.Equal(appStoreEvmRoot, evmStoreEvmRoot) {
		return nil, fmt.Errorf("EVM roots mismatch, version:%d, evm.db:%X, app.db:%X",
			appStore.Version(), evmStoreEvmRoot, appStoreEvmRoot)
	}
	store.setLastSavedTreeToVersion(appStore.Version())
	return store, nil
}

func (s *MultiWriterAppStore) Delete(key []byte) {
	if util.HasPrefix(key, vmPrefix) {
		s.evmStore.Delete(key)
		if !s.isEvmDBEnabled() {
			s.appStore.Delete(key)
		}
	} else {
		s.appStore.Delete(key)
	}
}

func (s *MultiWriterAppStore) Set(key, val []byte) {
	if util.HasPrefix(key, vmPrefix) {
		s.evmStore.Set(key, val)
		if !s.isEvmDBEnabled() {
			s.appStore.Set(key, val)
		}
	} else {
		s.appStore.Set(key, val)
	}
}

func (s *MultiWriterAppStore) Has(key []byte) bool {
	if util.HasPrefix(key, vmPrefix) && s.isEvmDBEnabled() {
		return s.evmStore.Has(key)
	}
	return s.appStore.Has(key)
}

func (s *MultiWriterAppStore) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) && s.isEvmDBEnabled() {
		return s.evmStore.Get(key)
	}
	return s.appStore.Get(key)
}

// Range iterates in-order over the keys in the store prefixed by the given prefix.
func (s *MultiWriterAppStore) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	if bytes.Equal(prefix, vmPrefix) && s.isEvmDBEnabled() {
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
		lvs := []string{"error", fmt.Sprint(err != nil)}
		saveVersionDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	currentRoot := s.evmStore.Commit(s.Version() + 1)
	if s.isEvmDBEnabled() {
		// Tie up Patricia tree with IAVL tree.
		// Do this after the feature flag is enabled so that we can detect
		// inconsistency in evm.db across the cluster
		s.appStore.Set(rootKey, currentRoot)
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
	// TODO: Need to ensure that the EvmStore and ImmutableTree are from the same height.
	appStoreTree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
	evmDbSnapshot := s.evmStore.GetSnapshot(appStoreTree.Version())
	featureKey := util.PrefixKey(featureKey, evmDBFeature)
	featureFlag := false
	_, data := appStoreTree.Get(featureKey)
	if bytes.Equal(data, []byte{1}) {
		featureFlag = true
	}
	evmStoreEnabled := s.evmStoreEnabled || featureFlag
	return NewMultiWriterStoreSnapshot(evmDbSnapshot, appStoreTree, evmStoreEnabled)
}

func (s *MultiWriterAppStore) isEvmDBEnabled() bool {
	featureKey := util.PrefixKey(featureKey, evmDBFeature)
	featureFlag := false
	data := s.appStore.Get(featureKey)
	if bytes.Equal(data, []byte{1}) {
		featureFlag = true
	}
	return s.evmStoreEnabled || featureFlag
}

type multiWriterStoreSnapshot struct {
	evmDbSnapshot   db.Snapshot
	appStoreTree    *iavl.ImmutableTree
	evmStoreEnabled bool
}

func NewMultiWriterStoreSnapshot(
	evmDbSnapshot db.Snapshot, appStoreTree *iavl.ImmutableTree, evmStoreEnabled bool,
) *multiWriterStoreSnapshot {
	return &multiWriterStoreSnapshot{
		evmDbSnapshot:   evmDbSnapshot,
		appStoreTree:    appStoreTree,
		evmStoreEnabled: evmStoreEnabled,
	}
}

func (s *multiWriterStoreSnapshot) Release() {
	s.evmDbSnapshot.Release()
	s.appStoreTree = nil
}

func (s *multiWriterStoreSnapshot) Has(key []byte) bool {
	if s.evmStoreEnabled && util.HasPrefix(key, vmPrefix) {
		return s.evmDbSnapshot.Has(key)
	}
	return s.appStoreTree.Has(key)
}

func (s *multiWriterStoreSnapshot) Get(key []byte) []byte {
	if s.evmStoreEnabled && util.HasPrefix(key, vmPrefix) {
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

	if s.evmStoreEnabled && bytes.Equal(prefix, vmPrefix) {
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
