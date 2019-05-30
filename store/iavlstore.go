package store

import (
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
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	pruneTime               metrics.Histogram
	iavlSaveVersionDuration metrics.Histogram
)

func init() {
	const namespace = "loomchain"
	const subsystem = "iavl_store"

	pruneTime = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "prune_duration",
			Help:      "How long IAVLStore.Prune() took to execute (in seconds)",
		}, []string{"error"},
	)
	iavlSaveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "save_version",
			Help:      "How long IAVLStore.SaveVersion() took to execute (in seconds)",
		}, []string{},
	)
}

var _ = VersionedKVStore(&IAVLStore{})

type IAVLStore struct {
	tree          *iavl.MutableTree
	lastSavedTree unsafe.Pointer // *iavl.ImmutableTree
	maxVersions   int64          // maximum number of versions to keep when pruning
}

func (s *IAVLStore) Delete(key []byte) {
	s.tree.Remove(key)
}

func (s *IAVLStore) Set(key, val []byte) {
	s.tree.Set(key, val)
}

func (s *IAVLStore) Has(key []byte) bool {
	return s.tree.Has(key)
}

func (s *IAVLStore) Get(key []byte) []byte {
	_, val := s.tree.Get(key)
	return val
}

// Returns the bytes that mark the end of the key range for the given prefix.
func prefixRangeEnd(prefix []byte) []byte {
	if prefix == nil {
		return nil
	}

	end := make([]byte, len(prefix))
	copy(end, prefix)

	for {
		if end[len(end)-1] != byte(255) {
			end[len(end)-1]++
			break
		} else if len(end) == 1 {
			end = nil
			break
		}
		end = end[:len(end)-1]
	}
	return end
}

func (s *IAVLStore) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	keys, values, _, err := s.tree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
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

		} else {
			continue // Skip this key as it does not have the prefix
		}

		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *IAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version()
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	var err error
	defer func(begin time.Time) {
		iavlSaveVersionDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	oldVersion := s.Version()
	hash, version, err := s.tree.SaveVersion()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
	}
	if err := s.setLastSavedTreeToVersion(version); err != nil {
		return nil, 0, err
	}
	return hash, version, nil
}

func (s *IAVLStore) Prune() error {
	// keep all the versions
	if s.maxVersions == 0 {
		return nil
	}

	latestVer := s.Version()
	oldVer := latestVer - s.maxVersions
	if oldVer < 1 {
		return nil
	}

	var err error
	defer func(begin time.Time) {
		lvs := []string{"error", fmt.Sprint(err != nil)}
		pruneTime.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	if s.tree.VersionExists(oldVer) {
		if err = s.tree.DeleteVersion(oldVer); err != nil {
			return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
		}
	}
	return nil
}

func (s *IAVLStore) GetSnapshot() Snapshot {
	return &iavlStoreSnapshot{
		ImmutableTree: (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree)),
	}
}

func (s *IAVLStore) setLastSavedTreeToVersion(version int64) error {
	var err error
	var tree *iavl.ImmutableTree

	if version == 0 {
		tree = iavl.NewImmutableTree(nil, 0)
	} else {
		tree, err = s.tree.GetImmutable(version)
		if err != nil {
			return errors.Wrapf(err, "failed to load immutable tree for version %v", version)
		}
	}

	atomic.StorePointer(&s.lastSavedTree, unsafe.Pointer(tree))
	return nil
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion int64) (*IAVLStore, error) {
	tree := iavl.NewMutableTree(db, 10000)
	_, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	// always keep at least 2 of the last versions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	return &IAVLStore{
		tree:        tree,
		maxVersions: maxVersions,
	}, nil
}

type iavlStoreSnapshot struct {
	*iavl.ImmutableTree
}

func (s *iavlStoreSnapshot) Get(key []byte) []byte {
	_, val := s.ImmutableTree.Get(key)
	return val
}

func (s *iavlStoreSnapshot) Has(key []byte) bool {
	return s.ImmutableTree.Has(key)
}

func (s *iavlStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	keys, values, _, err := s.ImmutableTree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
	if err != nil {
		log.Error("failed to get range", "err", err)
		panic(err)
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

		} else {
			continue // Skip this key as it does not have the prefix
		}
		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *iavlStoreSnapshot) Release() {
	s.ImmutableTree = nil
}
