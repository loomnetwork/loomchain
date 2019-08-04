package store

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	loom "github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	separator = "|"
	//Prometheus metrics
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

func init() {
	const namespace = "loomchain"
	const subsystem = "versioned_caching_store"

	getDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "get",
			Help:       "How long Versioned CachingStore.Get() took to execute (in miliseconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error", "isCacheHit"})

	hasDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "has",
			Help:       "How long Versioned CachingStore.Has() took to execute (in miliseconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error", "isCacheHit"})

	deleteDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "delete",
			Help:       "How long Versioned CachingStore.Delete() took to execute (in miliseconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"})

	setDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "set",
			Help:       "How long Versioned CachingStore.Set() took to execute (in miliseconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
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
			Help:      "number of errors encountered while doing any operation on cache",
		}, []string{"cache_operation"})

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

type versionedBigCache struct {
	cache         *bigcache.BigCache
	cacheLogger   *loom.Logger
	keyTableMutex sync.RWMutex
	keyTable      map[string]KeyVersionTable
}

func newVersionedBigCache(config *CachingStoreConfig, cacheLogger *loom.Logger) (*versionedBigCache, error) {
	bigcacheConfig, err := convertToBigCacheConfig(config, cacheLogger)
	if err != nil {
		return nil, err
	}
	versionedCache := &versionedBigCache{
		cacheLogger: cacheLogger,
		keyTable:    map[string]KeyVersionTable{},
	}

	// when a key get evicted from BigCache, KeyVersionTable and KeyTable must be updated
	bigcacheConfig.OnRemove = versionedCache.onRemove

	cache, err := bigcache.NewBigCache(*bigcacheConfig)
	if err != nil {
		return nil, err
	}
	versionedCache.cache = cache
	return versionedCache, nil
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

func (c *versionedBigCache) onRemove(key string, entry []byte) {
	c.keyTableMutex.Lock()
	defer c.keyTableMutex.Unlock()
	key, version, err := unversionedKey(key)
	if err != nil {
		c.cacheLogger.Error(fmt.Sprintf(
			"[VersionedBigCache] error while unversioning key: %s, error: %v",
			string(key), err.Error()))
	}
	kvTable, exist := c.keyTable[key]
	if exist {
		// remove all previous versions of the key
		for k, exist := range kvTable {
			if exist && k <= version {
				delete(kvTable, version)
			}
		}
		if len(kvTable) == 0 {
			delete(c.keyTable, key)
		}
	}
}

func (c *versionedBigCache) Delete(key []byte, version int64) error {
	versionedKey := versionedKey(string(key), version)
	// delete data in cache if it does exist
	c.cache.Delete(string(versionedKey))
	// add key to inidicate that this is the latest version but
	// the data has been deleted
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedBigCache) Set(key, val []byte, version int64) error {
	versionedKey := versionedKey(string(key), version)
	err := c.cache.Set(string(versionedKey), val)
	if err != nil {
		return err
	}
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedBigCache) Get(key []byte, version int64) ([]byte, error) {
	latestVersion := c.getKeyVersion(key, version)
	versionedKey := versionedKey(string(key), latestVersion)
	return c.cache.Get(string(versionedKey))
}

// getKeyVersion returns the latest version number (limited by version argument) of a particular key
func (c *versionedBigCache) getKeyVersion(key []byte, version int64) int64 {
	c.keyTableMutex.RLock()
	defer c.keyTableMutex.RUnlock()
	kvTable, exist := c.keyTable[string(key)]
	if !exist {
		return 0
	}
	var latestVersion int64
	for k, exists := range kvTable {
		if k > latestVersion && exists && k <= version {
			latestVersion = k
		}
	}
	return latestVersion
}

// addKeyVersion adds version number of a key to KeyVersionTable
func (c *versionedBigCache) addKeyVersion(key []byte, version int64) {
	c.keyTableMutex.Lock()
	defer c.keyTableMutex.Unlock()
	kvTable, exist := c.keyTable[string(key)]
	if !exist {
		kvTable = KeyVersionTable{}
	}
	kvTable[version] = true
	c.keyTable[string(key)] = kvTable
}

// versionedCachingStore wraps a write-through cache around a VersionedKVStore.
// It is compatible with MultiWriterAppStore only.
type versionedCachingStore struct {
	VersionedKVStore
	cache   *versionedBigCache
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

	versionedBigCache, err := newVersionedBigCache(config, cacheLogger)
	if err != nil {
		return nil, err
	}

	return &versionedCachingStore{
		VersionedKVStore: source,
		cache:            versionedBigCache,
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
	cache   *versionedBigCache
	version int64
	logger  *loom.Logger
}

func newVersionedCachingStoreSnapshot(snapshot Snapshot, cache *versionedBigCache,
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
