package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type IAVLStore struct {
	tree        *iavl.VersionedTree
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

func (s *IAVLStore) Range(prefix []byte) plugin.RangeData {
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

func (s *IAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version64()
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
	if s.tree.VersionExists(oldVer) {
		if err := s.tree.DeleteVersion(oldVer); err != nil {
			return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
		}
	}
	return nil
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
func NewIAVLStore(db dbm.DB, maxVersions int64) (*IAVLStore, error) {
	tree := iavl.NewVersionedTree(db, 10000)
	_, err := tree.Load()
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
