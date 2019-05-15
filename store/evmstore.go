package store

import (
	"bytes"
	"encoding/binary"
	"sort"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/pkg/errors"
)

var (
	defaultRoot = []byte{1}
)

func evmRootKey(blockHeight int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(blockHeight))
	return util.PrefixKey(vmPrefix, []byte(evmRootPrefix), b)
}

type EvmStore struct {
	evmDB db.DBWrapper
	cache map[string]cacheItem
}

func NewEvmStore(evmDB db.DBWrapper) *EvmStore {
	evmStore := &EvmStore{
		evmDB: evmDB,
	}
	evmStore.Rollback()
	return evmStore
}

func (s *EvmStore) setCache(key, val []byte, deleted bool) {
	s.cache[string(key)] = cacheItem{
		Value:   val,
		Deleted: deleted,
	}
}

func (s *EvmStore) Range(prefix []byte) plugin.RangeData {
	rangeCacheKeys := []string{}
	rangeCache := make(map[string][]byte)

	// Add records from evm.db to range cache
	iter := s.evmDB.Iterator(prefix, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := iter.Value()
		if util.HasPrefix([]byte(key), prefix) || len(prefix) == 0 {
			rangeCache[key] = value
			rangeCacheKeys = append(rangeCacheKeys, key)
		}
	}

	// Update range cache with data in cache
	for key, c := range s.cache {
		if util.HasPrefix([]byte(key), prefix) || len(prefix) == 0 {
			if c.Deleted {
				rangeCacheKeys = remove(rangeCacheKeys, key)
				rangeCache[key] = nil
				continue
			}
			if _, ok := rangeCache[key]; !ok {
				rangeCacheKeys = append(rangeCacheKeys, string(key))
			}
			rangeCache[key] = c.Value
		}
	}

	ret := make(plugin.RangeData, 0)
	// Sorting makes RangeData deterministic
	sort.Strings(rangeCacheKeys)
	for _, key := range rangeCacheKeys {
		var unprefixedKey []byte
		var err error
		if len(prefix) > 0 {
			unprefixedKey, err = util.UnprefixKey([]byte(key), prefix)
			if err != nil {
				continue
			}
		} else {
			unprefixedKey = []byte(key)
		}
		re := &plugin.RangeEntry{
			Key:   unprefixedKey,
			Value: rangeCache[key],
		}
		ret = append(ret, re)
	}
	return ret
}

func (s *EvmStore) Has(key []byte) bool {
	if item, ok := s.cache[string(key)]; ok {
		return !item.Deleted
	}
	return s.evmDB.Has(key)
}

func (s *EvmStore) Get(key []byte) []byte {
	if item, ok := s.cache[string(key)]; ok {
		return item.Value
	}
	return s.evmDB.Get(key)
}

func (s *EvmStore) Delete(key []byte) {
	s.setCache(key, nil, true)
}

func (s *EvmStore) Set(key, val []byte) {
	s.setCache(key, val, false)
}

func (s *EvmStore) Commit(version int64) []byte {
	// Save versioning roots
	currentRoot := s.Get(util.PrefixKey(vmPrefix, rootKey))
	if currentRoot == nil {
		currentRoot = defaultRoot
	}
	s.Set(evmRootKey(version), currentRoot)

	batch := s.evmDB.NewBatch()
	for key, item := range s.cache {
		if !item.Deleted {
			batch.Set([]byte(key), item.Value)
		} else {
			batch.Delete([]byte(key))
		}
	}
	batch.Write()
	s.Rollback()
	return currentRoot
}

func (s *EvmStore) LoadVersion(targetVersion int64) error {
	s.Rollback()
	// To ensure that evm root is corresponding with iavl tree height,
	// copy current root of Patricia tree to vmvmroot in evm.db.
	// LoomEthDb uses vmvmroot as a key to current root of Patricia tree.
	root := s.evmDB.Get(evmRootKey(targetVersion))
	if root == nil && targetVersion != 0 {
		return errors.Errorf("failed to load EVM root for version %d", targetVersion)
		// []byte{1} root inidicates that the Patricia tree is empty at targetVersion,
		// so we need to set vmroot to empty
	} else if bytes.Equal(root, defaultRoot) {
		root = []byte("")
	}
	s.evmDB.Set(util.PrefixKey(vmPrefix, rootKey), root)
	return nil
}

func (s *EvmStore) Rollback() {
	s.cache = make(map[string]cacheItem)
}

func (s *EvmStore) GetSnapshot() db.Snapshot {
	return s.evmDB.GetSnapshot()
}

func remove(keys []string, key string) []string {
	for i, value := range keys {
		if value == key {
			return append(keys[:i], keys[i+1:]...)
		}
	}
	return keys
}
