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
	tree             *iavl.MutableTree
	maxVersions      int64 // maximum number of versions to keep when pruning
	saveFrequency    uint64
	versionFrequency uint64
	saveCount        uint64
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
	defer func(err error) {
		if r := recover(); r != nil {
			if err == nil {
				err = errors.Errorf("panic in SaveVersion %v", r)
			} else {
				err = errors.Wrapf(err, "panic in SaveVersion %v", r)
			}

		}
	}(err)
	//log.Info("database size", "size", s.tree.Size())
	oldVersion := s.Version()
	if s.saveFrequency > 0 {
		s.saveCount++
		if s.saveCount%s.saveFrequency != 0 {
			hash, version, err := s.tree.NewVersion()
			if err != nil {
				return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
			}

			return hash, version, nil
		} else {
			s.saveCount = 0
		}
	}

	hash, version, err := s.tree.SaveVersion()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
	}

	return hash, version, nil
}

func (s *IAVLStore) Prune() error {
	var err error
	defer func(err error) {
		if r := recover(); r != nil {
			if err == nil {
				err = errors.Errorf("panic in DeleteVersion %v", r)
			} else {
				err = errors.Wrapf(err, "panic in DeleteVersion %v", r)
			}

		}
	}(err)
	// keep all the versions
	if s.maxVersions == 0 {
		return nil
	}

	latestVer := s.Version()
	oldVer := latestVer - s.maxVersions
	if oldVer < 1 {
		return nil
	}

	if s.versionFrequency != 0 && uint64(oldVer)%s.versionFrequency == 0 {
		return nil
	}

	defer func(begin time.Time) {
		lvs := []string{"error", fmt.Sprint(err != nil)}
		pruneTime.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	if s.tree.VersionExists(oldVer) {
		if s.saveFrequency == 0 {
			if err = s.tree.DeleteVersion(oldVer); err != nil {
				return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
			}
		} else {
			if err = s.tree.DeleteMemoryVersion(oldVer, true); err != nil {
				return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
			}
		}
	}
	return nil
}

func (s *IAVLStore) GetSnapshot() Snapshot {
	// This isn't an actual snapshot obviously, and never will be, but lets pretend...
	return &iavlStoreSnapshot{
		IAVLStore: s,
	}
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
// saveFrequency says how often the IVAL tree will be saved to the disk. 0 means every block.
// versionFrequency = N, indicates that versions other than multiples of N will be eventually pruned.
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion int64, saveFrequency, versionFrequency uint64) (*IAVLStore, error) {
	ndb := iavl.NewNodeDB3(db, 10000, saveFrequency > 0, nil)
	tree := iavl.NewMutableTreeWithNodeDB(ndb)
	_, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	// always keep at least 2 of the last versions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	return &IAVLStore{
		tree:             tree,
		maxVersions:      maxVersions,
		saveFrequency:    saveFrequency,
		versionFrequency: versionFrequency,
	}, nil
}

type iavlStoreSnapshot struct {
	*IAVLStore
}

func (s *iavlStoreSnapshot) Release() {
	// noop
}
