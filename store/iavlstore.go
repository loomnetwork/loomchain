package store

import (
	"bytes"
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
	pruneTime               metrics.Histogram
	iavlSaveVersionDuration metrics.Histogram
)

func init() {
	const namespace = "loomchain"
	const subsystem = "iavl_store"

	pruneTime = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "prune_duration",
			Help:       "How long IAVLStore.Prune() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"},
	)
	iavlSaveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "save_version",
			Help:       "How long IAVLStore.SaveVersion() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)
}

type IAVLStore struct {
	tree          *iavl.MutableTree
	maxVersions   int64 // maximum number of versions to keep when pruning
	flushInterval int64 // how often we persist to disk
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
	return s.RangeWithLimit(prefix, 0)
}

// RangeWithLimit will return a list of keys & values that are prefixed by the given bytes (with a
// zero byte separator between the prefix and the key).
//
// If the limit is zero all matching keys will be returned, if the limit is greater than zero at most
// that many keys will be returned. Unfortunately, specifying a non-zero limit can result in somewhat
// unpredictable results, if there are N matching keys, and the limit is N, the number of keys
// returned may be less than N.
func (s *IAVLStore) RangeWithLimit(prefix []byte, limit int) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	keys, values, _, err := s.tree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), limit)
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
	flushInterval := s.flushInterval

	// TODO: Rather than loading the on-chain config here the flush interval override should be passed
	//       in as a parameter to SaveVersion().
	if flushInterval == 0 {
		cfg, err := LoadOnChainConfig(s)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to load on-chain config")
		}
		if cfg.GetAppStore().GetIAVLFlushInterval() != 0 {
			flushInterval = int64(cfg.GetAppStore().GetIAVLFlushInterval())
		}
	} else if flushInterval == -1 {
		flushInterval = 0
	}

	var version int64
	var hash []byte
	// Every X versions we should persist to disk
	if flushInterval == 0 || ((oldVersion+1)%flushInterval == 0) {
		if flushInterval != 0 {
			log.Info("[IAVLStore] Flushing mem to disk", "version", oldVersion+1)
			hash, version, err = s.tree.FlushMemVersionDisk()
		} else {
			hash, version, err = s.tree.SaveVersion()
		}
	} else {
		hash, version, err = s.tree.SaveVersionMem()
	}

	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
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
	snapshot, err := s.GetSnapshotAt(0)
	if err != nil {
		panic(err)
	}
	return snapshot
}

func (s *IAVLStore) GetSnapshotAt(version int64) (Snapshot, error) {
	return newIavlStoreSnapshot(*s, version)
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
// flushInterval specifies the number of IAVL tree versions that should be kept in memory before
// writing a new version to disk. If set to zero every version will be written to disk unless overriden
// via the on-chain config. If set to -1 every version will always be written to disk, regardless of
// the on-chain config.
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion, flushInterval int64) (*IAVLStore, error) {
	if flushInterval < -1 {
		return nil, errors.New("invalid flush interval")
	}

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
		tree:          tree,
		maxVersions:   maxVersions,
		flushInterval: flushInterval,
	}, nil
}

type iavlStoreSnapshot struct {
	*iavl.ImmutableTree
}

func newIavlStoreSnapshot(store IAVLStore, version int64) (Snapshot, error) {
	if version == 0 {
		version = store.Version()
	}
	immutableTree, err := store.tree.GetImmutable(version)
	if err != nil {
		return nil, err
	}
	return &iavlStoreSnapshot{
		immutableTree,
	}, nil
}

func (s *iavlStoreSnapshot) Get(key []byte) []byte {
	_, value := s.ImmutableTree.Get(key)
	return value
}

func (s *iavlStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	var data plugin.RangeData
	prefix = append(prefix, 0)
	s.ImmutableTree.IterateRangeInclusive(prefix, nil, true, func(key []byte, value []byte, _ int64) bool {
		if len(key) < len(prefix) || 0 != bytes.Compare(prefix, key[:len(prefix)]) {
			return true
		}
		data = append(data, &plugin.RangeEntry{key, value})
		return false
	})
	return data
}

func (s *iavlStoreSnapshot) Release() {}
