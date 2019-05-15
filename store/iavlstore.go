package store

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
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
	tree        *iavl.MutableTree
	maxVersions int64 // maximum number of versions to keep when pruning
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

func UnprefixKey2(key, prefix []byte) ([]byte, error) {
	if len(prefix) > len(key) {
		return nil, fmt.Errorf("prefix2 %s longer than key %s", string(prefix), string(key))
	}
	return key[len(prefix):], nil
}

func (s *IAVLStore) Range(prefix []byte) plugin.RangeData {
	fmt.Printf("IAVL-Range-%v\n", prefix)
	fmt.Printf("IAVL-Range--%s\n", string(prefix))
	ret := make(plugin.RangeData, 0)
	if bytes.IndexAny(prefix, "delegation") > -1 {
		fmt.Printf("has suffix delegation\n")
		return s.Range2(prefix)
	}
	fmt.Printf("Doesn't have suffix delegation\n")
	end := prefixRangeEnd(prefix)
	fmt.Printf("end-%s\n", end)

	keys, values, _, err := s.tree.GetRangeWithProof(prefix, end, 0)
	fmt.Printf("Found %d --- KEYS!!! in Range\n", len(keys))
	if err != nil {
		fmt.Printf("failed to get range  err -%v", err)
		return ret
	}
	for i, x := range keys {
		fmt.Printf("raw key-%s\n", x)
		k, err := util.UnprefixKey(x, prefix)
		if err != nil {
			fmt.Printf("raw key-%s\n", x)
			fmt.Printf("failed to unprefix key -%s prefix -%s err-%v", x, prefix, err)
			k = nil
		}
		fmt.Printf("\nunprefixed key-%s\n", k)

		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *IAVLStore) Range2(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	end := prefixRangeEnd(prefix)
	//end = []byte("delegation3")
	fmt.Printf("end-%s\n", end)

	var keys [][]byte
	var values [][]byte
	fn := func(key, value []byte) bool {
		fmt.Printf("Key-%s value -%s\n", string(key), string(value))
		keys = append(keys, key)
		values = append(values, value)
		return false
	}
	s.tree.IterateRange(prefix, end, true, fn)
	fmt.Printf("Found(range2) %d --- KEYS!!! in Range\n", len(keys))
	for i, x := range keys {
		//TODO return this to greatness
		k, err := UnprefixKey2(x, prefix)
		if err != nil {
			fmt.Printf("failed to unprefix key -%s prefix -%s err-%v\n", x, prefix, err)
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
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version()
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	oldVersion := s.Version()
	hash, version, err := s.tree.SaveVersion()
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
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion int64) (*IAVLStore, error) {
	tree := iavl.NewMutableTree(db, 1000000)
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
	*IAVLStore
}

func (s *iavlStoreSnapshot) Release() {
	// noop
}
