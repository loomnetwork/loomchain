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

type IAVLStore struct {
	// TODO: ugh, having two trees is messy, should split this out into MutableIAVLStore & ImmutableIAVLStore,
	//       with each one containing a single mutable or immutable tree.
	mutableTree   *iavl.MutableTree   // Only set in the mutable store
	immutableTree *iavl.ImmutableTree // Set in both the mutable & immutable store
	maxVersions   int64               // maximum number of versions to keep when pruning
}

func (s *IAVLStore) Delete(key []byte) {
	s.mutableTree.Remove(key)
}

func (s *IAVLStore) Set(key, val []byte) {
	s.mutableTree.Set(key, val)
}

func (s *IAVLStore) Has(key []byte) bool {
	return s.immutableTree.Has(key)
}

func (s *IAVLStore) Get(key []byte) []byte {
	_, val := s.immutableTree.Get(key)
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

	keys, values, _, err := s.immutableTree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
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

func (s *IAVLStore) Hash() []byte {
	return s.immutableTree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.immutableTree.Version()
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	if s.mutableTree == nil {
		return nil, 0, errors.New("Can't save new version in immutable store")
	}
	oldVersion := s.Version()
	hash, version, err := s.mutableTree.SaveVersion()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
	}
	s.immutableTree = s.mutableTree.ImmutableTree
	return hash, version, nil
}

func (s *IAVLStore) GetImmutableVersion(version int64) (VersionedKVStore, error) {
	if s.mutableTree == nil {
		return nil, errors.New("Can't load another version in immutable store")
	}
	t, err := s.mutableTree.GetImmutable(version)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load immutable tree for version %v", version)
	}
	return &IAVLStore{
		mutableTree:   nil,
		immutableTree: t,
		maxVersions:   s.maxVersions,
	}, nil
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
		mutableTree:   tree,
		immutableTree: tree.ImmutableTree,
		maxVersions:   maxVersions,
	}, nil
}
