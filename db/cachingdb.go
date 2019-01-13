package db

import (
	"fmt"
	"math"
	"time"

	"github.com/allegro/bigcache"

	"github.com/go-kit/kit/metrics"

	"github.com/loomnetwork/loomchain/log"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const FlushInterval = 1 * time.Minute

var (
	getDuration    metrics.Histogram
	hasDuration    metrics.Histogram
	deleteDuration metrics.Histogram
	setDuration    metrics.Histogram

	cacheHits   metrics.Counter
	cacheErrors metrics.Counter
	cacheMisses metrics.Counter
)

type CachingDBLogger struct {
}

func (c CachingDBLogger) Printf(format string, v ...interface{}) {
	log.Default.Info(format, v)
}

type CachingDBConfig struct {
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
	const subsystem = "caching_db"

	getDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "get",
			Help:      "How long CachingDB.Get() took to execute (in miliseconds)",
		}, []string{"error", "isCacheHit"})

	hasDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "has",
			Help:      "How long CachingDB.Has() took to execute (in miliseconds)",
		}, []string{"error", "isCacheHit"})

	deleteDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "delete",
			Help:      "How long CachingDB.Delete() took to execute (in miliseconds)",
		}, []string{"error"})

	setDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "set",
			Help:      "How long CachingDB.Set() took to execute (in miliseconds)",
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

// CachingDB wraps a cache around a DBWrapperWithBatch.
type CachingDB struct {
	DBWrapperWithBatch
	cache  *bigcache.BigCache
	quitCh chan struct{}
}

func DefaultCachingDBConfig() *CachingDBConfig {
	return &CachingDBConfig{
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

func convertToBigCacheConfig(config *CachingDBConfig) (*bigcache.Config, error) {
	if config.MaxKeys == 0 || config.MaxSizeOfValueInBytes == 0 {
		return nil, fmt.Errorf("[CachingDB] max keys and/or max size of value cannot be zero")
	}

	if config.EvictionTimeInSeconds == 0 {
		return nil, fmt.Errorf("[CachingDB] eviction time cannot be zero")
	}

	if config.Shards == 0 {
		return nil, fmt.Errorf("[CachingDB] caching shards cannot be zero")
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
	configTemplate.Logger = CachingDBLogger{}

	return &configTemplate, nil
}

func NewCachingDB(source DBWrapperWithBatch, config *CachingDBConfig) (*CachingDB, error) {
	if config == nil {
		return nil, fmt.Errorf("[CachingDB] config cant be null for caching store")
	}

	bigcacheConfig, err := convertToBigCacheConfig(config)
	if err != nil {
		return nil, err
	}

	cache, err := bigcache.NewBigCache(*bigcacheConfig)
	if err != nil {
		return nil, err
	}

	cachingDB := &CachingDB{
		DBWrapperWithBatch: source,
		cache:              cache,
		quitCh:             make(chan struct{}),
	}

	cachingDB.startFlushRoutine()

	return cachingDB, nil
}

func (c *CachingDB) Shutdown() {
	log.Error("[CachingStore] Flushed Writebatch due to quitting")
	c.DBWrapperWithBatch.FlushBatch()
}

func (c *CachingDB) startFlushRoutine() {
	go func() {
		timer := time.NewTicker(FlushInterval)
		for {
			<-timer.C
			c.DBWrapperWithBatch.FlushBatch()
			log.Error("[CachingStore] Flushed Writebatch due to timer")
		}
	}()
}

func (c *CachingDB) DeleteSync(key []byte) {
	var err error

	defer func(begin time.Time) {
		deleteDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Delete(string(key))
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "deleteSync").Add(1)
		log.Error(fmt.Sprintf("[CachingDB] error while deleting key: %s in cache, error: %v", string(key), err.Error()))
	}

	c.DBWrapperWithBatch.BatchDelete(key)
}

func (c *CachingDB) Delete(key []byte) {
	var err error

	defer func(begin time.Time) {
		deleteDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Delete(string(key))
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "delete").Add(1)
		log.Error(fmt.Sprintf("[CachingDB] error while deleting key: %s in cache, error: %v", string(key), err.Error()))
	}

	c.DBWrapperWithBatch.BatchDelete(key)
}

func (c *CachingDB) SetSync(key, val []byte) {
	var err error

	defer func(begin time.Time) {
		setDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Set(string(key), val)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "setSync").Add(1)
		log.Error(fmt.Sprintf("[CachingDB] error while setting key: %s in cache, error: %v", string(key), err.Error()))
	}

	c.DBWrapperWithBatch.BatchSet(key, val)
}

func (c *CachingDB) Set(key, val []byte) {
	var err error

	defer func(begin time.Time) {
		setDuration.With("error", fmt.Sprint(err != nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	err = c.cache.Set(string(key), val)
	if err != nil {
		// Only log error and dont error out
		cacheErrors.With("cache_operation", "set").Add(1)
		log.Error(fmt.Sprintf("[CachingDB] error while setting key: %s in cache, error: %v", string(key), err.Error()))
	}

	c.DBWrapperWithBatch.BatchSet(key, val)
}

func (c *CachingDB) Get(key []byte) []byte {
	var err error

	defer func(begin time.Time) {
		getDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(string(key))

	if err != nil {
		cacheMisses.With("store_operation", "get").Add(1)
		switch e := err.(type) {
		case *bigcache.EntryNotFoundError:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			log.Error(fmt.Sprintf("[CachingDB] error while getting key: %s from cache, error: %v", string(key), e.Error()))
		}

		data = c.DBWrapperWithBatch.Get(key)
		if data == nil {
			return nil
		}
		setErr := c.cache.Set(string(key), data)
		if setErr != nil {
			cacheErrors.With("cache_operation", "set").Add(1)
			log.Error(fmt.Sprintf("[CachingDB] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
		}
	} else {
		cacheHits.With("store_operation", "get").Add(1)
	}

	return data
}

func (c *CachingDB) Has(key []byte) bool {
	var err error

	defer func(begin time.Time) {
		hasDuration.With("error", fmt.Sprint(err != nil), "isCacheHit", fmt.Sprint(err == nil)).Observe(float64(time.Since(begin).Nanoseconds()) / math.Pow10(6))
	}(time.Now())

	data, err := c.cache.Get(string(key))
	exists := true

	if err != nil {
		cacheMisses.With("store_operation", "has").Add(1)
		switch e := err.(type) {
		case *bigcache.EntryNotFoundError:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source and only log the error
			cacheErrors.With("cache_operation", "get").Add(1)
			log.Error(fmt.Sprintf("[CachingDB] error while getting key: %s from cache, error: %v", string(key), e.Error()))
		}

		data = c.DBWrapperWithBatch.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(string(key), data)
			if setErr != nil {
				cacheErrors.With("cache_operation", "set").Add(1)
				log.Error(fmt.Sprintf("[CachingDB] error while setting key: %s in cache, error: %v", string(key), setErr.Error()))
			}
		}
	} else {
		cacheHits.With("store_operation", "has").Add(1)
	}

	return exists
}
