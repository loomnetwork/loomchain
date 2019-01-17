package store

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/allegro/bigcache"

	"github.com/go-kit/kit/metrics"

	"github.com/loomnetwork/loomchain/log"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	getDuration    metrics.Histogram
	hasDuration    metrics.Histogram
	deleteDuration metrics.Histogram
	setDuration    metrics.Histogram

	cacheHits   metrics.Counter
	cacheErrors metrics.Counter
	cacheMisses metrics.Counter
)

type CachingStoreLogger struct {
}

func (c CachingStoreLogger) Printf(format string, v ...interface{}) {
	log.Default.Info(format, v)
}

type CachingStoreConfig struct {
	CachingEnabled bool
	// Number of cache shards, value must be a power of two
	Shards int
	// Time after we need to evict the key
	EvictionTimeInSeconds int64
	// interval at which clean up of expired keys will occur
	CleaningIntervalInSeconds int64
	// Total size of cache would be: MaxKeys*MaxSizeOfValueInBytes
	MaxKeys               int
	MaxSizeOfValueInBytes int

	// Logs operations
	Verbose bool
}

func init() {
	const namespace = "loomchain"
	const subsystem = "caching_store"

	getDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "get",
			Help:      "How long CachingStore.Get() took to execute (in miliseconds)",
		}, []string{"error", "isCacheHit"})

	hasDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "has",
			Help:      "How long CachingStore.Has() took to execute (in miliseconds)",
		}, []string{"error", "isCacheHit"})

	deleteDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "delete",
			Help:      "How long CachingStore.Delete() took to execute (in miliseconds)",
		}, []string{"error"})

	setDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "set",
			Help:      "How long CachingStore.Set() took to execute (in miliseconds)",
		}, []string{"error"})

	cacheHits = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hit",
			Help:      "Number of cache hit for get/has",
		}, []string{"store_operation"})

	cacheMisses = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_miss",
			Help:      "Number of cache miss for get/has",
		}, []string{"store_operation"})

	cacheErrors = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_error",
			Help:      "number of errors enocuntered while doing any operation on cache",
		}, []string{"cache_operation"})

}

// CachingStore wraps a write-through cache around a VersionedKVStore.
// NOTE: Writes update the cache, reads do not, to read from the cache use the store returned by
//       ReadOnly().
type CachingStore struct {
	VersionedKVStore
	cache *bigcache.BigCache
}

func DefaultCachingStoreConfig() *CachingStoreConfig {
	return &CachingStoreConfig{
		CachingEnabled:            true,
		Shards:                    1024,
		EvictionTimeInSeconds:     60 * 60, // 1 hour
		CleaningIntervalInSeconds: 10,      // Cleaning per 10 second
		// Approximately 110 MB
		MaxKeys:               50 * 10 * 100,
		MaxSizeOfValueInBytes: 2048,
		Verbose:               true,
	}
}

func convertToBigCacheConfig(config *CachingStoreConfig) (*bigcache.Config, error) {
	if config.MaxKeys == 0 || config.MaxSizeOfValueInBytes == 0 {
		return nil, fmt.Errorf("[CachingStore] max keys and/or max size of value cannot be zero")
	}

	if config.EvictionTimeInSeconds == 0 {
		return nil, fmt.Errorf("[CachingStore] eviction time cannot be zero")
	}

	if config.Shards == 0 {
		return nil, fmt.Errorf("[CachingStore] caching shards cannot be zero")
	}

	configTemplate := bigcache.DefaultConfig(time.Duration(config.EvictionTimeInSeconds) * time.Second)
	configTemplate.Shards = config.Shards
	configTemplate.Verbose = config.Verbose
	configTemplate.CleanWindow = time.Duration(config.CleaningIntervalInSeconds) * time.Second
	configTemplate.LifeWindow = time.Duration(config.EvictionTimeInSeconds) * time.Second
	configTemplate.HardMaxCacheSize = config.MaxKeys * config.MaxSizeOfValueInBytes
	configTemplate.MaxEntriesInWindow = config.MaxKeys
	configTemplate.MaxEntrySize = config.MaxSizeOfValueInBytes
	configTemplate.Verbose = config.Verbose
	configTemplate.Logger = CachingStoreLogger{}

	return &configTemplate, nil
}

func NewCachingStore(source VersionedKVStore, config *CachingStoreConfig) (*CachingStore, error) {
	if config == nil {
		return nil, fmt.Errorf("[CachingStore] config cant be null for caching store")
	}

	bigcacheConfig, err := convertToBigCacheConfig(config)
	if err != nil {
		return nil, err
	}

	cache, err := bigcache.NewBigCache(*bigcacheConfig)
	if err != nil {
		return nil, err
	}

	return &CachingStore{
		VersionedKVStore: source,
		cache:            cache,
	}, nil
}

func (c *CachingStore) Delete(key []byte) {
	var err error

	defer func(begin time.Time) {
		deleteDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Delete(string(key))
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "delete").Add(1)
		log.Error(fmt.Sprintf("[CachingStore] error while deleting key: %s in cache, error: %v", string(key), err.Error()))
	}
	c.VersionedKVStore.Delete(key)
}

func (c *CachingStore) Set(key, val []byte) {
	var err error

	defer func(begin time.Time) {
		setDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Set(string(key), val)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "set").Add(1)
		log.Error(fmt.Sprintf("[CachingStore] error while setting key: %s in cache, error: %v", string(key), err.Error()))
	}
	c.VersionedKVStore.Set(key, val)
}

// ReadOnlyCachingStore prevents any modification to the underlying backing store,
// and uses the cache for reads.
type ReadOnlyCachingStore struct {
	*CachingStore
}

func NewReadOnlyCachingStore(cachingStore *CachingStore) *ReadOnlyCachingStore {
	return &ReadOnlyCachingStore{
		CachingStore: cachingStore,
	}
}

func (c *ReadOnlyCachingStore) Delete(key []byte) {
	panic("[ReadOnlyCachingStore] Delete() not implemented")
}

func (c *ReadOnlyCachingStore) Set(key, val []byte) {
	panic("[ReadOnlyCachingStore] Set() not implemented")
}

func (c *ReadOnlyCachingStore) Has(key []byte) bool {
	var err error

	defer func(begin time.Time) {
		hasDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(string(key))
	exists := true

	if err != nil {
		cacheMisses.With("store_operation", "has").Add(1)
		switch err {
		case bigcache.ErrEntryNotFound:
			log.Debug(fmt.Sprintf("[ReadOnlyCachingStore][CacheMiss] key: string: %s, Hex: %x Operation: %s", string(key), key, "Has"))
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			log.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data = c.VersionedKVStore.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(string(key), data)
			if setErr != nil {
				cacheErrors.With("cache_operation", "set").Add(1)
				log.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
			}
		}
	} else {
		log.Debug(fmt.Sprintf("[ReadOnlyCachingStore][CacheHit] key: string: %s, Hex: %x Operation: %s", string(key), key, "Has"))
		cacheHits.With("store_operation", "has").Add(1)
	}

	return exists
}

func (c *ReadOnlyCachingStore) Get(key []byte) []byte {
	var err error

	defer func(begin time.Time) {
		getDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(string(key))

	if err != nil {
		cacheMisses.With("store_operation", "get").Add(1)
		switch err {
		case bigcache.ErrEntryNotFound:
			log.Debug(fmt.Sprintf("[ReadOnlyCachingStore][CacheMiss] key: string: %s, Hex: %x  Operation: %s", string(key), key, "Get"))
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			log.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data = c.VersionedKVStore.Get(key)
		if data == nil {
			return nil
		}
		setErr := c.cache.Set(string(key), data)
		if setErr != nil {
			cacheErrors.With("cache_operation", "set").Add(1)
			log.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
		}
	} else {
		log.Debug(fmt.Sprintf("[ReadOnlyCachingStore][CacheHit] key: string: %s, Hex: %x Operation: %s", string(key), key, "Get"))
		cacheHits.With("store_operation", "get").Add(1)
	}

	return data
}

func (c *ReadOnlyCachingStore) SaveVersion() ([]byte, int64, error) {
	return nil, 0, errors.New("[ReadOnlyCachingStore] SaveVersion() not implemented")
}

func (c *ReadOnlyCachingStore) Prune() error {
	return errors.New("[ReadOnlyCachingStore] Prune() not implemented")
}
