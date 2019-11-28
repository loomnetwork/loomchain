package store

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/allegro/bigcache"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	loom "github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const separator = "|"

// Prometheus metrics
var (
	getDuration    metrics.Histogram
	hasDuration    metrics.Histogram
	deleteDuration metrics.Histogram
	setDuration    metrics.Histogram

	cacheHits   metrics.Counter
	cacheErrors metrics.Counter
	cacheMisses metrics.Counter
)

type versionedCache interface {
	Get(key []byte, version int64) ([]byte, error)
	Set(key, val []byte, version int64) error
	Delete(key []byte, version int64) error
}

type CachingStoreLogger struct {
	logger *loom.Logger
}

func (c CachingStoreLogger) Printf(format string, v ...interface{}) {
	c.logger.Info(format, v)
}

func init() {
	const namespace = "loomchain"
	const subsystem = "caching_store"

	getDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "get",
			Help:       "How long VersionedCachingStore.Get() took to execute",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error", "isCacheHit"})

	hasDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "has",
			Help:       "How long VersionedCachingStore.Has() took to execute",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error", "isCacheHit"})

	deleteDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "delete",
			Help:       "How long VersionedCachingStore.Delete() took to execute",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"})

	setDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "set",
			Help:       "How long VersionedCachingStore.Set() took to execute",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"})

	cacheHits = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hit",
			Help:      "Number of cache hits for VersionCachingStore.Get/Has",
		}, []string{"store_operation"})

	cacheMisses = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_miss",
			Help:      "Number of cache miss for VersionCachingStore.Get/Has",
		}, []string{"store_operation"})

	cacheErrors = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_error",
			Help:      "Number of errors encountered while doing any operation on cache",
		}, []string{"cache_operation"})

}

type CachingStoreConfig struct {
	// 0 = disabled, 1 = bigCache, 2 = fastCache
	CachingType int
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

func DefaultCachingStoreConfig() *CachingStoreConfig {
	return &CachingStoreConfig{
		CachingType:               0,
		Shards:                    1024,
		EvictionTimeInSeconds:     60 * 60,       // 1 hour
		CleaningIntervalInSeconds: 10,            // Cleaning per 10 second
		MaxKeys:                   50 * 10 * 100, // Approximately 110 MB
		MaxSizeOfValueInBytes:     2048,
		Verbose:                   true,
		LogDestination:            "file://-",
		LogLevel:                  "info",
	}
}

// KeyVersionTable keeps versions of a cached key
type KeyVersionTable map[int64]bool

func versionedKey(key string, version int64) string {
	v := strconv.FormatInt(version, 10)
	return v + separator + string(key)
}

func unversionedKey(key string) (string, int64, error) {
	k := strings.SplitN(key, separator, 2)
	if len(k) != 2 {
		return "", 0, fmt.Errorf("Invalid versioned key %s", string(key))
	}
	n, err := strconv.ParseInt(k[0], 10, 64)
	if err != nil {
		return "", 0, err
	}
	return k[1], n, nil
}

// versionedCachingStore wraps a write-through cache around a VersionedKVStore.
// It is compatible with MultiWriterAppStore only.
type versionedCachingStore struct {
	VersionedKVStore
	cache   versionedCache
	version int64
	logger  *loom.Logger
}

// NewVersionedCachingStore wraps the source VersionedKVStore in a cache.
func NewVersionedCachingStore(
	source VersionedKVStore, config *CachingStoreConfig, version int64,
) (VersionedKVStore, error) {
	if config == nil {
		return nil, fmt.Errorf("[VersionedCachingStore] missing config for caching store")
	}

	cacheLogger := loom.NewLoomLogger(config.LogLevel, config.LogDestination)

	var cache versionedCache
	var err error
	switch config.CachingType {
	case 0: // disabled
		return source, nil
	case 1:
		cache, err = newVersionedBigCache(config, cacheLogger)
	case 2:
		cache, err = newVersionedFastCache(config, cacheLogger)
	default:
		return nil, fmt.Errorf("unrecognised cahcing type %v", config.CachingType)
	}
	if err != nil {
		return nil, err
	}

	return &versionedCachingStore{
		VersionedKVStore: source,
		cache:            cache,
		logger:           cacheLogger,
		version:          version + 1,
	}, nil
}

func (c *versionedCachingStore) Delete(key []byte) {
	var err error

	defer func(begin time.Time) {
		deleteDuration.With("error",
			fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Delete(key, c.version)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "delete").Add(1)
		c.logger.Error(fmt.Sprintf(
			"[VersionedCachingStore] error while deleting key: %s in cache, error: %v",
			string(key), err.Error()))
	}
	c.VersionedKVStore.Delete(key)
}

func (c *versionedCachingStore) Set(key, val []byte) {
	var err error

	defer func(begin time.Time) {
		setDuration.With("error",
			fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Set(key, val, c.version)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "set").Add(1)
		c.logger.Error(fmt.Sprintf(
			"[VersionedCachingStore] error while setting key: %s in cache, error: %v",
			string(key), err.Error()))
	}
	c.VersionedKVStore.Set(key, val)
}

func (c *versionedCachingStore) SaveVersion() ([]byte, int64, error) {
	hash, version, err := c.VersionedKVStore.SaveVersion()
	if err == nil {
		// Cache version is always 1 block ahead of KV store version, that way when
		// GetSnapshot() is called it won't return the current unpersisted state of the cache,
		// but rather the last persisted version.
		c.version = version + 1
	}
	return hash, version, err
}

func (c *versionedCachingStore) GetSnapshot() Snapshot {
	return newVersionedCachingStoreSnapshot(
		c.VersionedKVStore.GetSnapshot(),
		c.cache, c.version-1, c.logger,
	)
}

// CachingStoreSnapshot is a read-only CachingStore with specified version
type versionedCachingStoreSnapshot struct {
	Snapshot
	cache   versionedCache
	version int64
	logger  *loom.Logger
}

func newVersionedCachingStoreSnapshot(snapshot Snapshot, cache versionedCache,
	version int64, logger *loom.Logger) *versionedCachingStoreSnapshot {
	return &versionedCachingStoreSnapshot{
		Snapshot: snapshot,
		cache:    cache,
		version:  version,
		logger:   logger,
	}
}

func (c *versionedCachingStoreSnapshot) Delete(key []byte) {
	panic("[versionedCachingStoreSnapshot] Delete() not implemented")
}

func (c *versionedCachingStoreSnapshot) Set(key, val []byte) {
	panic("[versionedCachingStoreSnapshot] Set() not implemented")
}

func (c *versionedCachingStoreSnapshot) Has(key []byte) bool {
	var err error

	defer func(begin time.Time) {
		hasDuration.With("error",
			fmt.Sprint(err != nil),
			"isCacheHit",
			fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	_, err = c.cache.Get(key, c.version)
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
			c.logger.Error(fmt.Sprintf(
				"[versionedCachingStoreSnapshot] error while getting key: %s from cache, error: %v",
				string(key), err.Error()))
		}

		data := c.Snapshot.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(key, data, c.version)
			if setErr != nil {
				cacheErrors.With("cache_operation", "set").Add(1)
				c.logger.Error(fmt.Sprintf(
					"[versionedCachingStoreSnapshot] error while setting key: %s in cache, error: %v",
					string(key), setErr.Error()))
			}
		}
	} else {
		cacheHits.With("store_operation", "has").Add(1)
	}

	return exists
}

func (c *versionedCachingStoreSnapshot) Get(key []byte) []byte {
	var err error
	defer func(begin time.Time) {
		getDuration.With("error", fmt.Sprint(err != nil), "isCacheHit",
			fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(key, c.version)

	if err != nil {
		cacheMisses.With("store_operation", "get").Add(1)
		switch err {
		case bigcache.ErrEntryNotFound:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			c.logger.Error(fmt.Sprintf(
				"[versionedCachingStoreSnapshot] error while getting key: %s from cache, error: %v",
				string(key), err.Error()))
		}

		data = c.Snapshot.Get(key)
		setErr := c.cache.Set(key, data, c.version)
		if setErr != nil {
			cacheErrors.With("cache_operation", "set").Add(1)
			c.logger.Error(fmt.Sprintf(
				"[versionedCachingStoreSnapshot] error while setting key: %s in cache, error: %v",
				string(key), setErr.Error()))
		}
	} else {
		cacheHits.With("store_operation", "get").Add(1)
	}

	return data
}

func (c *versionedCachingStoreSnapshot) SaveVersion() ([]byte, int64, error) {
	return nil, 0, errors.New("[VersionedCachingStoreSnapshot] SaveVersion() not implemented")
}

func (c *versionedCachingStoreSnapshot) Prune() error {
	return errors.New("[VersionedCachingStoreSnapshot] Prune() not implemented")
}

func (c *versionedCachingStoreSnapshot) Release() {
	c.Snapshot.Release()
}
