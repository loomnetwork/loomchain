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

func getVersionFromEvmRootKey(key []byte) (int64, error) {
	v, err := util.UnprefixKey(key, util.PrefixKey(vmPrefix, []byte(evmRootPrefix)))
	if err != nil {
		return 0, err
	}
	version := int64(binary.BigEndian.Uint64(v))
	return version, nil
}

// EvmStore persists EVM state to a DB.
type EvmStore struct {
	evmDB db.DBWrapper
	cache map[string]cacheItem
}

// NewEvmStore returns a new instance of the store backed by the given DB.
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

// Range iterates in-order over the keys in the store prefixed by the given prefix.
// TODO (VM): This needs a proper review, other than tests there is no code that really makes use of
//            this function, only place it's called is from MultiWriterAppStore.Range but only when
//            iterating over the "vm" prefix - which no code currently does.
// NOTE: This version of EvmStore supports Range(nil)
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
	currentRoot := s.Get(util.PrefixKey(vmPrefix, rootKey))
	latestSavedRoot, _ := s.getLatestSavedRoot(0)
	// default root is an indicator for empty root
	if bytes.Equal(currentRoot, []byte("")) {
		currentRoot = defaultRoot
	}
	// save Patricia root of EVM state only if it changes
	if !bytes.Equal(currentRoot, latestSavedRoot) {
		s.Set(evmRootKey(version), currentRoot)
	}

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
	// find the lastest saved root
	root, _ := s.getLatestSavedRoot(targetVersion)
	if bytes.Equal(root, defaultRoot) {
		root = []byte("")
	}

	// nil root indicates that latest saved root below target version is not found
	if root == nil && targetVersion > 0 {
		return errors.Errorf("failed to load EVM root for version %d", targetVersion)
	}

	// To ensure that evm root is corresponding with iavl tree height,
	// store current root of Patricia tree (root hash) in EvmStore cache.
	// LoomEthDb uses vmvmroot as a key to current root of Patricia tree
	// TODO: This needs to be refactored to avoid writing to the DB on load.
	s.Set(util.PrefixKey(vmPrefix, rootKey), root)
	return nil
}

func (s *EvmStore) Rollback() {
	s.cache = make(map[string]cacheItem)
}

func (s *EvmStore) getLatestSavedRoot(targetVersion int64) ([]byte, int64) {
	iter := s.evmDB.ReverseIterator(util.PrefixKey(vmPrefix, evmRootPrefix), nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		if util.HasPrefix(iter.Key(), util.PrefixKey(vmPrefix, evmRootPrefix)) {
			version, err := getVersionFromEvmRootKey(iter.Key())
			if err != nil {
				return nil, 0
			}
			if version <= targetVersion || targetVersion == 0 {
				return iter.Value(), version
			}
		}
	}
	return nil, 0
}

func (s *EvmStore) GetSnapshot(version int64) db.Snapshot {
	targetRoot, _ := s.getLatestSavedRoot(version)
	return NewEvmStoreSnapshot(s.evmDB.GetSnapshot(), targetRoot)
}

func NewEvmStoreSnapshot(snapshot db.Snapshot, rootHash []byte) *EvmStoreSnapshot {
	return &EvmStoreSnapshot{
		Snapshot: snapshot,
		rootHash: rootHash,
	}
}

type EvmStoreSnapshot struct {
	db.Snapshot
	rootHash []byte
}

func (s *EvmStoreSnapshot) Get(key []byte) []byte {
	if bytes.Equal(key, util.PrefixKey(vmPrefix, rootKey)) {
		return s.rootHash
	}
	return s.Snapshot.Get(key)
}

func (s *EvmStoreSnapshot) Has(key []byte) bool {
	// snapshot always has a root hash
	// nil or empty root hash is considered valid root hash
	if bytes.Equal(key, util.PrefixKey(vmPrefix, rootKey)) {
		return true
	}
	return s.Snapshot.Has(key)
}

func remove(keys []string, key string) []string {
	for i, value := range keys {
		if value == key {
			return append(keys[:i], keys[i+1:]...)
		}
	}
	return keys
}
