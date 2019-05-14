package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/allegro/bigcache"
	"github.com/gogo/protobuf/proto"

	"github.com/go-kit/kit/metrics"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"

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

	versionTablePrefix = []byte("versiontable")
)

func versionedKey(key []byte, version int64) []byte {
	return util.PrefixKey(key, int64ToBytes(version))
}

func versionTableKey(key []byte) []byte {
	return util.PrefixKey(versionTablePrefix, key)
}

type VersionedBigCache struct {
	cache  *bigcache.BigCache
	logger *loom.Logger
}

func NewVersionedBigCache(cache *bigcache.BigCache, logger *loom.Logger) *VersionedBigCache {
	return &VersionedBigCache{
		cache:  cache,
		logger: logger,
	}
}

func (c *VersionedBigCache) Delete(key []byte, version int64) error {
	var err error
	versionedKey := versionedKey(key, version)
	// delete data in cache if it does exist
	c.cache.Delete(string(versionedKey))
	// add key to inidicate that this is the latest version but
	// the data has been delete
	if err := c.addKeyVersion(key, version); err != nil {
		return err
	}
	return err
}

func (c *VersionedBigCache) Set(key, val []byte, version int64) error {
	versionedKey := versionedKey(key, version)
	err := c.cache.Set(string(versionedKey), val)
	if err != nil {
		return err
	}
	err = c.addKeyVersion(key, version)
	if err != nil {
		return err
	}
	return nil
}

func (c *VersionedBigCache) Has(key []byte, version int64) bool {
	latestVersion := c.getKeyVersion(key, version)
	versionedKey := versionedKey(key, latestVersion)
	data, err := c.cache.Get(string(versionedKey))
	exists := true
	if err != nil || data == nil {
		exists = false
	}
	return exists
}

func (c *VersionedBigCache) Get(key []byte, version int64) ([]byte, error) {
	latestVersion := c.getKeyVersion(key, version)
	versionedKey := versionedKey(key, latestVersion)
	data, err := c.cache.Get(string(versionedKey))
	if err != nil {
		return data, err
	}
	if data == nil {
		err = bigcache.ErrEntryNotFound
	}
	return data, err
}

func (c *VersionedBigCache) getKeyVersion(key []byte, version int64) int64 {
	tableKey := versionTableKey(key)
	var kt KeyVersionTable
	var latestVersion int64
	buf, err := c.cache.Get(string(tableKey))
	if err != nil {
		return latestVersion
	}
	if err := proto.Unmarshal(buf, &kt); err != nil {
		return latestVersion
	}
	for k, exists := range kt.Keys {
		if k > latestVersion && exists && k <= version {
			latestVersion = k
		}
	}
	return latestVersion
}

func (c *VersionedBigCache) addKeyVersion(key []byte, version int64) error {
	tableKey := versionTableKey(key)
	var kt KeyVersionTable
	// get key table
	buf, err := c.cache.Get(string(tableKey))
	if err != nil {
		if err != bigcache.ErrEntryNotFound {
			return err
		}
		kt = KeyVersionTable{
			Keys: map[int64]bool{},
		}
	} else {
		if err := proto.Unmarshal(buf, &kt); err != nil {
			return err
		}
	}

	kt.Keys[version] = true

	//save key table
	buf, err = proto.Marshal(&kt)
	if err != nil {
		return err
	}
	if err := c.cache.Set(string(tableKey), buf); err != nil {
		return err
	}
	return nil
}

func (c *VersionedBigCache) deleteKeyVersion(key []byte, version int64) error {
	tableKey := versionTableKey(key)
	var kt KeyVersionTable
	// get key table
	buf, err := c.cache.Get(string(tableKey))
	if err != nil {
		if err != bigcache.ErrEntryNotFound {
			return err
		}
		kt = KeyVersionTable{
			Keys: map[int64]bool{},
		}
	} else {
		if err := proto.Unmarshal(buf, &kt); err != nil {
			return err
		}
	}

	kt.Keys[version] = false
	// save key table
	buf, err = proto.Marshal(&kt)
	if err != nil {
		return err
	}
	if err := c.cache.Set(string(tableKey), buf); err != nil {
		return err
	}

	return nil
}

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

func (c *CachingStore) Delete(key []byte) {
	var err error

	defer func(begin time.Time) {
		deleteDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Delete(key, c.version)
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

	err = c.cache.Set(key, val, c.version)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "set").Add(1)
		c.logger.Error(fmt.Sprintf("[CachingStore] error while setting key: %s in cache, error: %v", string(key), err.Error()))
	}
	c.VersionedKVStore.Set(key, val)
}

// CachingStore wraps a write-through cache around a VersionedKVStore.
// NOTE: Writes update the cache, reads do not, to read from the cache use the store returned by
//       ReadOnly().
type CachingStore struct {
	VersionedKVStore
	cache   *VersionedBigCache
	version int64
	logger  *loom.Logger
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
	configTemplate.Logger = CachingStoreLogger{logger: logger}

	return &configTemplate, nil
}

func NewCachingStore(source VersionedKVStore, config *CachingStoreConfig, version int64) (*CachingStore, error) {
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

	versionedBigCache := NewVersionedBigCache(cache, cacheLogger)

	return &CachingStore{
		VersionedKVStore: source,
		cache:            versionedBigCache,
		logger:           cacheLogger,
		version:          version,
	}, nil
}

func (c *CachingStore) SaveVersion() ([]byte, int64, error) {
	hash, version, err := c.VersionedKVStore.SaveVersion()
	if err == nil {
		c.version = version + 1
	}
	return hash, version, err
}

func (c *CachingStore) GetSnapshot() Snapshot {
	kvStoreSnapshot := c.VersionedKVStore.GetSnapshot()
	return NewCachingStoreSnapshot(kvStoreSnapshot, c.cache, c.version, c.logger)
}

// CachingStoreSnapshot is a read-only CachingStore with specified version
type CachingStoreSnapshot struct {
	Snapshot
	cache   *VersionedBigCache
	version int64
	logger  *loom.Logger
}

func NewCachingStoreSnapshot(snapshot Snapshot, cache *VersionedBigCache, version int64, logger *loom.Logger) *CachingStoreSnapshot {
	return &CachingStoreSnapshot{
		Snapshot: snapshot,
		cache:    cache,
		version:  version,
		logger:   logger,
	}
}

func (c *CachingStoreSnapshot) Delete(key []byte) {
	panic("[CachingStoreSnapshot] Delete() not implemented")
}

func (c *CachingStoreSnapshot) Set(key, val []byte) {
	panic("[CachingStoreSnapshot] Set() not implemented")
}

func (c *CachingStoreSnapshot) Has(key []byte) bool {
	var err error

	defer func(begin time.Time) {
		hasDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
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
			c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data := c.Snapshot.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(key, data, c.version)
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

func (c *CachingStoreSnapshot) Get(key []byte) []byte {
	var err error
	defer func(begin time.Time) {
		getDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
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
			c.logger.Error(fmt.Sprintf("[CachingStoreSnapshot] error while getting key: %s from cache, error: %v", string(key), err.Error()))
		}

		data = c.Snapshot.Get(key)
		setErr := c.cache.Set(key, data, c.version)
		if setErr != nil {
			cacheErrors.With("cache_operation", "set").Add(1)
			c.logger.Error(fmt.Sprintf("[ReadOnlyCachingStore] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
		}
	} else {
		cacheHits.With("store_operation", "get").Add(1)
	}

	return data
}

func (c *CachingStoreSnapshot) SaveVersion() ([]byte, int64, error) {
	return nil, 0, errors.New("[CachingStoreSnapshot] SaveVersion() not implemented")
}

func (c *CachingStoreSnapshot) Prune() error {
	return errors.New("[CachingStoreSnapshot] Prune() not implemented")
}

func (c *CachingStoreSnapshot) Release() {
	c.Snapshot.Release()
}

func int64ToBytes(n int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(n))
	return buf
}
