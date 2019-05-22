package store

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/allegro/bigcache"

	"github.com/go-kit/kit/metrics"

	loom "github.com/loomnetwork/go-loom"

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
	logger *loom.Logger
}

func (c CachingStoreLogger) Printf(format string, v ...interface{}) {
	c.logger.Info(format, v)
}

type CachingStoreConfig struct {
	CachingEnabled bool
	// CachingEnabled may be ignored in some configurations, this will force enable the caching
	// store in those cases.
	// WARNING: This should only used for debugging.
	DebugForceEnable bool
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

	LogLevel       string
	LogDestination string
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
		LogDestination:        "file://-",
		LogLevel:              "info",
	}
}

func convertToBigCacheConfig(config *CachingStoreConfig, logger *loom.Logger) (*bigcache.Config, error) {
	if config.MaxKeys == 0 || config.MaxSizeOfValueInBytes == 0 {
		return nil, fmt.Errorf("[CachingStoreConfig] max keys and/or max size of value cannot be zero")
	}

	if config.EvictionTimeInSeconds == 0 {
		return nil, fmt.Errorf("[CachingStoreConfig] eviction time cannot be zero")
	}

	if config.Shards == 0 {
		return nil, fmt.Errorf("[CachingStoreConfig] caching shards cannot be zero")
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
	configTemplate.Logger = CachingStoreLogger{logger: logger}

	return &configTemplate, nil
}

// CachingStore wraps a write-through cache around a VersionedKVStore.
// NOTE: Writes update the cache, reads do not, to read from the cache use the store returned by
//       ReadOnly().
type CachingStore struct {
	VersionedKVStore
	cache  *bigcache.BigCache
	logger *loom.Logger
}

func NewCachingStore(source VersionedKVStore, config *CachingStoreConfig) (*CachingStore, error) {
	if config == nil {
		return nil, fmt.Errorf("[CachingStore] config cant be null for caching store")
	}

	cacheLogger := loom.NewLoomLogger(config.LogLevel, config.LogDestination)

	bigcacheConfig, err := convertToBigCacheConfig(config, cacheLogger)
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
		logger:           cacheLogger,
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
		c.logger.Error(fmt.Sprintf("[CachingStore] error while deleting key: %s in cache, error: %v", string(key), err.Error()))
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
		c.logger.Error(fmt.Sprintf("[CachingStore] error while setting key: %s in cache, error: %v", string(key), err.Error()))
	}
	c.VersionedKVStore.Set(key, val)
}

func (c *CachingStore) GetSnapshot() Snapshot {
	return newReadOnlyCachingStore(c)
}

// ReadOnlyCachingStore prevents any modification to the underlying backing store,
// and uses the cache for reads.
type readOnlyCachingStore struct {
	*CachingStore
}

func newReadOnlyCachingStore(cachingStore *CachingStore) *readOnlyCachingStore {
	return &readOnlyCachingStore{
		CachingStore: cachingStore,
	}
}

func (c *readOnlyCachingStore) Delete(key []byte) {
	panic("[ReadOnlyCachingStore] Delete() not implemented")
}

func (c *readOnlyCachingStore) Set(key, val []byte) {
	panic("[ReadOnlyCachingStore] Set() not implemented")
}

func (c *readOnlyCachingStore) Has(key []byte) bool {
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
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data = c.VersionedKVStore.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(string(key), data)
			if setErr != nil {
				cacheErrors.With("cache_operation", "set").Add(1)
				c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
			}
		}
	} else {
		cacheHits.With("store_operation", "has").Add(1)
	}

	return exists
}

func (c *readOnlyCachingStore) Get(key []byte) []byte {
	var err error

	defer func(begin time.Time) {
		getDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(string(key))

	if err != nil {
		cacheMisses.With("store_operation", "get").Add(1)
		switch err {
		case bigcache.ErrEntryNotFound:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data = c.VersionedKVStore.Get(key)
		if data == nil {
			return nil
		}
		setErr := c.cache.Set(string(key), data)
		if setErr != nil {
			cacheErrors.With("cache_operation", "set").Add(1)
			c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
		}
	} else {
		cacheHits.With("store_operation", "get").Add(1)
	}

	return data
}

func (c *readOnlyCachingStore) SaveVersion() ([]byte, int64, error) {
	return nil, 0, errors.New("[ReadOnlyCachingStore] SaveVersion() not implemented")
}

func (c *readOnlyCachingStore) Prune() error {
	return errors.New("[ReadOnlyCachingStore] Prune() not implemented")
}

// Implements Snapshot interface
func (c *readOnlyCachingStore) Release() {
	// noop
}
