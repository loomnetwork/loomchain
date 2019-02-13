package store

import (
	"fmt"
	"time"

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
	pruneTime metrics.Histogram
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
		}, []string{"error"})

}

type ImmutableIAVLStore struct {
	tree *iavl.ImmutableTree
}

func (s *ImmutableIAVLStore) Delete(key []byte) {
	panic("Can't delete in ImmutableIAVLStore")
}

func (s *ImmutableIAVLStore) Set(key, val []byte) {
	panic("Can't set in ImmutableIAVLStore")
}

func (s *ImmutableIAVLStore) Has(key []byte) bool {
	return s.tree.Has(key)
}

func (s *ImmutableIAVLStore) Get(key []byte) []byte {
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

func (s *ImmutableIAVLStore) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	keys, values, _, err := s.tree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
	if err != nil {
		log.Error("failed to get range", "err", err)
		return ret
	}
	for i, x := range keys {
		k, err := util.UnprefixKey(x, prefix)
		if err != nil {
			log.Error("failed to unprefix key", "key", x, "prefix", prefix, "err", err)
			k = nil
		}
		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *ImmutableIAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *ImmutableIAVLStore) Version() int64 {
	return s.tree.Version()
}

func (s *ImmutableIAVLStore) SaveVersion() ([]byte, int64, error) {
	return nil, 0, errors.New("Can't save ImmutableIAVLStore")
}

func (s *ImmutableIAVLStore) ReadOnly() VersionedKVStore {
	return s
}

func (s *ImmutableIAVLStore) Prune() error {
	panic("Can't prune ImmutableIAVLStore")
}

type IAVLStore struct {
	*ImmutableIAVLStore
	mutableTree   *iavl.MutableTree
	maxVersions   int64 // maximum number of versions to keep when pruning
	mvcc          bool
	readOnlyStore *ImmutableIAVLStore
}

func (s *IAVLStore) Delete(key []byte) {
	s.mutableTree.Remove(key)
}

func (s *IAVLStore) Set(key, val []byte) {
	s.mutableTree.Set(key, val)
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	oldVersion := s.Version()
	hash, version, err := s.mutableTree.SaveVersion()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
	}
	s.ImmutableIAVLStore.tree = s.mutableTree.ImmutableTree

	if s.mvcc {
		if err := s.setReadOnlyStoreToVersion(version); err != nil {
			return nil, 0, err
		}
	}

	return hash, version, nil
}

func (s *IAVLStore) setReadOnlyStoreToVersion(version int64) error {
	tree, err := s.mutableTree.GetImmutable(version)
	if err != nil {
		return errors.Wrapf(err, "failed to load immutable tree for version %v", version)
	}

	// Load the whole tree into memory to avoid hitting the disk when accessing it from here on out
	tree.Preload()

	s.readOnlyStore = &ImmutableIAVLStore{
		tree: tree,
	}

	return nil
}

func (s *IAVLStore) ReadOnly() VersionedKVStore {
	if s.readOnlyStore != nil {
		return s.readOnlyStore
	}
	return s
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

	if s.mutableTree.VersionExists(oldVer) {
		if err = s.mutableTree.DeleteVersion(oldVer); err != nil {
			return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
		}
	}
	return nil
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion int64, enableMVCC bool) (*IAVLStore, error) {
	tree := iavl.NewMutableTree(db, 10000)
	latestVer, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	// always keep at least 2 of the last versions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	store := &IAVLStore{
		ImmutableIAVLStore: &ImmutableIAVLStore{
			tree: tree.ImmutableTree,
		},
		mutableTree: tree,
		maxVersions: maxVersions,
		mvcc:        enableMVCC,
	}

	if enableMVCC {
		if err := store.setReadOnlyStoreToVersion(latestVer); err != nil {
			return nil, err
		}
	}

	return store, nil
}
