package store

import (
	"bytes"
	"sort"
	"sync/atomic"
	"unsafe"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
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
)

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

func (s *MultiWriterAppStore) Range(prefix []byte) plugin.RangeData {
	appStoreRange := s.appStore.Range(prefix)
	if s.isEvmDBEnabled() {
		cache := make(map[string][]byte)
		keys := []string{}
		for _, data := range appStoreRange {
			cache[string(data.Key)] = data.Value
			keys = append(keys, string(data.Key))
		}

		// Theoretically data in evm.db is newer than data in app.db once
		// EvmDBFeature has been activated
		evmStoreRange := s.evmStore.Range(prefix)
		for _, data := range evmStoreRange {
			if _, ok := cache[string(data.Key)]; !ok {
				keys = append(keys, string(data.Key))
			}
			cache[string(data.Key)] = data.Value
		}

		ret := make(plugin.RangeData, 0)
		sort.Strings(keys)
		for _, k := range keys {
			ret = append(ret, &plugin.RangeEntry{
				Key:   []byte(k),
				Value: cache[k],
			})
		}

		appStoreRange = ret
	}
	return appStoreRange
}

func (s *MultiWriterAppStore) Hash() []byte {
	return s.appStore.Hash()
}

func (s *MultiWriterAppStore) Version() int64 {
	return s.appStore.Version()
}

func (s *MultiWriterAppStore) SaveVersion() ([]byte, int64, error) {
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
	evmDbSnapshot := s.evmStore.GetSnapshot()
	appStoreTree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
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

func NewMultiWriterStoreSnapshot(evmDbSnapshot db.Snapshot, appStoreTree *iavl.ImmutableTree, evmStoreEnabled bool) *multiWriterStoreSnapshot {
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
	if util.HasPrefix(key, vmPrefix) && s.evmStoreEnabled {
		return s.evmDbSnapshot.Has(key)
	}
	return s.appStoreTree.Has(key)
}

func (s *multiWriterStoreSnapshot) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) && s.evmStoreEnabled {
		return s.evmDbSnapshot.Get(key)
	}
	_, val := s.appStoreTree.Get(key)
	return val
}

func (s *multiWriterStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	rangeCacheKeys := []string{}
	rangeCache := make(map[string][]byte)
	ret := make(plugin.RangeData, 0)
	// Range from app.db snapshot
	keys, values, _, err := s.appStoreTree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
	if err != nil {
		log.Error("failed to get range", "err", err)
		return ret
	}
	for i, k := range keys {
		// Tree range gives all keys that has prefix but it does not check zero byte
		// after the prefix. So we have to check zero byte after prefix using util.HasPrefix
		if util.HasPrefix(k, prefix) {
			k, err = util.UnprefixKey(k, prefix)
			if err != nil {
				log.Error("failed to unprefix key", "key", k, "prefix", prefix, "err", err)
				k = nil
			}
			// If key does not have a prefix and prefix length > 0, skip this key
		} else if len(prefix) > 0 {
			continue
		}
		rangeCacheKeys = append(rangeCacheKeys, string(k))
		rangeCache[string(k)] = values[i]
	}

	// Range from evm.db snapshot
	it := s.evmDbSnapshot.NewIterator(prefix, prefixRangeEnd(prefix))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if util.HasPrefix(key, prefix) {
			key, err = util.UnprefixKey(key, prefix)
			if err != nil {
				log.Error("failed to unprefix key", "key", it.Key(), "prefix", prefix, "err", err)
				panic(err)
			}
		} else if len(prefix) > 0 {
			continue
		}
		if _, ok := rangeCache[string(key)]; !ok {
			rangeCacheKeys = append(rangeCacheKeys, string(key))
		}
		rangeCache[string(key)] = it.Value()
	}

	sort.Strings(rangeCacheKeys)
	for _, key := range rangeCacheKeys {
		re := &plugin.RangeEntry{
			Key:   []byte(key),
			Value: rangeCache[key],
		}
		ret = append(ret, re)
	}
	return ret
}
