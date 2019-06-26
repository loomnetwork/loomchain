package store

import (
	"bytes"
	"encoding/binary"
	"sort"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	lru "github.com/hashicorp/golang-lru"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	defaultRoot = []byte{1}
	rootHashKey = util.PrefixKey(vmPrefix, rootKey)

	commitDuration metrics.Histogram
)

func init() {
	commitDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  "loomchain",
			Subsystem:  "evmstore",
			Name:       "commit",
			Help:       "How long EvmStore.Commit() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)
}

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
	evmDB         db.DBWrapper
	cache         map[string]cacheItem
	rootHash      []byte
	lastSavedRoot []byte
	rootCache     *lru.Cache
	version       int64
}

// NewEvmStore returns a new instance of the store backed by the given DB.
func NewEvmStore(evmDB db.DBWrapper, numCachedRoots int) *EvmStore {
	rootCache, err := lru.New(numCachedRoots)
	if err != nil {
		panic(err)
	}
	evmStore := &EvmStore{
		evmDB:     evmDB,
		cache:     make(map[string]cacheItem),
		rootCache: rootCache,
	}
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

	// Make Range return root hash (vmvmroot) from EvmStore.rootHash
	if _, exist := rangeCache[string(rootHashKey)]; exist {
		rangeCache[string(rootHashKey)] = s.rootHash
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
	// EvmStore always has Patricia root
	if bytes.Equal(key, rootHashKey) {
		return true
	}
	if item, ok := s.cache[string(key)]; ok {
		return !item.Deleted
	}
	return s.evmDB.Has(key)
}

func (s *EvmStore) Get(key []byte) []byte {
	if bytes.Equal(key, rootHashKey) {
		return s.rootHash
	}

	if item, ok := s.cache[string(key)]; ok {
		return item.Value
	}
	return s.evmDB.Get(key)
}

func (s *EvmStore) Delete(key []byte) {
	if bytes.Equal(key, rootHashKey) {
		s.rootHash = nil
	} else {
		s.setCache(key, nil, true)
	}
}

func (s *EvmStore) Set(key, val []byte) {
	if bytes.Equal(key, rootHashKey) {
		s.rootHash = val
	} else {
		s.setCache(key, val, false)
	}
}

func (s *EvmStore) Commit(version int64) []byte {
	defer func(begin time.Time) {
		commitDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	currentRoot := make([]byte, len(s.rootHash))
	copy(currentRoot, s.rootHash)
	// default root is an indicator for empty root
	if bytes.Equal(currentRoot, []byte{}) {
		currentRoot = defaultRoot
	}
	// save Patricia root of EVM state only if it changes
	if !bytes.Equal(currentRoot, s.lastSavedRoot) {
		s.Set(evmRootKey(version), currentRoot)
	}

	s.rootCache.Add(version, currentRoot)

	batch := s.evmDB.NewBatch()
	for key, item := range s.cache {
		if !item.Deleted {
			batch.Set([]byte(key), item.Value)
		} else {
			batch.Delete([]byte(key))
		}
	}
	batch.Write()
	s.cache = make(map[string]cacheItem)
	s.lastSavedRoot = currentRoot
	s.version = version
	return currentRoot
}

func (s *EvmStore) LoadVersion(targetVersion int64) error {
	s.cache = make(map[string]cacheItem)
	// find the last saved root
	root, version := s.getLastSavedRoot(targetVersion)
	if bytes.Equal(root, defaultRoot) {
		root = []byte{}
	}
	s.rootCache.Add(targetVersion, root)

	// nil root indicates that latest saved root below target version is not found
	if root == nil && targetVersion > 0 {
		return errors.Errorf("failed to load EVM root for version %d", targetVersion)
	}

	s.rootHash = root
	s.lastSavedRoot = root
	s.version = version
	return nil
}

func (s *EvmStore) Version() ([]byte, int64) {
	return s.rootHash, s.version
}

func (s *EvmStore) getLastSavedRoot(targetVersion int64) ([]byte, int64) {
	start := util.PrefixKey(vmPrefix, evmRootPrefix)
	end := prefixRangeEnd(evmRootKey(targetVersion))
	iter := s.evmDB.ReverseIterator(start, end)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		version, err := getVersionFromEvmRootKey(iter.Key())
		if err != nil {
			return nil, 0
		}
		if version <= targetVersion || targetVersion == 0 {
			return iter.Value(), version
		}
	}
	return nil, 0
}

func (s *EvmStore) GetSnapshot(version int64) db.Snapshot {
	var targetRoot []byte
	// Expect cache to be almost 100% hit since cache miss yields extremely poor performance
	val, exist := s.rootCache.Get(version)
	if exist {
		targetRoot = val.([]byte)
	} else {
		targetRoot, _ = s.getLastSavedRoot(version)
	}
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
	if bytes.Equal(key, rootHashKey) {
		return s.rootHash
	}
	return s.Snapshot.Get(key)
}

func (s *EvmStoreSnapshot) Has(key []byte) bool {
	// snapshot always has a root hash
	// nil or empty root hash is considered valid root hash
	if bytes.Equal(key, rootHashKey) {
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
