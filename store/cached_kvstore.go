package store

import (
	"time"

	"github.com/allegro/bigcache"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	kvCacheHits   metrics.Counter
	kvCacheMisses metrics.Counter
)

func init() {
	const namespace = "loomchain"
	const subsystem = "caching_kv_store"
	kvCacheHits = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hit",
			Help:      "Number of cache hits for CachingKVStore.Get/Has",
		}, []string{"cache_operation"})

	kvCacheMisses = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_miss",
			Help:      "Number of cache miss for CachingKVStore.Get/Has",
		}, []string{"cache_operation"})
}

type CachingKVStore struct {
	VersionedKVStore
	cache *bigcache.BigCache
}

func NewCachingKVStore(store VersionedKVStore) (VersionedKVStore, error) {
	config := bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 1024,

		// time after which entry can be evicted
		LifeWindow: 10 * time.Minute,

		// Interval between removing expired entries (clean up).
		// If set to <= 0 then no action is performed.
		// Setting to < 1 second is counterproductive — bigcache has a one second resolution.
		CleanWindow: 5 * time.Minute,

		// rps * lifeWindow, used only in initial memory allocation
		MaxEntriesInWindow: 1000 * 10 * 60,

		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 500,

		// prints information about additional memory allocation
		Verbose: true,

		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 8192,

		// callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A bitmask representing the reason will be returned.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,

		// OnRemoveWithReason is a callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A constant representing the reason will be passed through.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		// Ignored if OnRemove is specified.
		OnRemoveWithReason: nil,
	}
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, err
	}
	return &CachingKVStore{
		VersionedKVStore: store,
		cache:            cache,
	}, nil
}

func (s *CachingKVStore) Delete(key []byte) {
	s.cache.Delete(string(key))
	s.VersionedKVStore.Delete(key)
}

func (s *CachingKVStore) Set(key, val []byte) {
	s.cache.Set(string(key), val)
	s.VersionedKVStore.Set(key, val)
}

func (s *CachingKVStore) Has(key []byte) bool {
	_, err := s.cache.Get(string(key))
	if err != nil {
		kvCacheMisses.With("cache_operation", "has").Add(1)
		val := s.VersionedKVStore.Get(key)
		has := len(val) > 0
		if has {
			s.cache.Set(string(key), val)
		}
		return has
	}
	kvCacheHits.With("cache_operation", "has").Add(1)
	return true
}

func (s *CachingKVStore) Get(key []byte) []byte {
	val, err := s.cache.Get(string(key))
	if err != nil {
		kvCacheMisses.With("cache_operation", "get").Add(1)
		val = s.VersionedKVStore.Get(key)
		s.cache.Set(string(key), val)
		return val
	}
	kvCacheHits.With("cache_operation", "get").Add(1)
	return val
}
