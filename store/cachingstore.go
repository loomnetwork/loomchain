package store

import (
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/loomnetwork/go-loom/plugin"

	"log"
)

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

type CachingStore struct {
	source VersionedKVStore
	cache  *bigcache.BigCache
}

func DefaultCachingStoreConfig() *CachingStoreConfig {
	return &CachingStoreConfig{
		CachingEnabled:            false,
		Shards:                    1024,
		EvictionTimeInSeconds:     60 * 60, // 1 hour
		CleaningIntervalInSeconds: 0,       // No cleaning
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
		source: source,
		cache:  cache,
	}, nil
}

func (c *CachingStore) Delete(key []byte) {
	err := c.cache.Delete(string(key))
	if err != nil {
		// Only log error and dont error out
		log.Printf("[CachingStore] error while deleting key in cache, error: %v\n", err.Error())
	}
	c.source.Delete(key)
}

func (c *CachingStore) Set(key, val []byte) {
	err := c.cache.Set(string(key), val)
	if err != nil {
		// Only log error and dont error out
		log.Printf("[CachingStore] error while setting key in cache, error: %v\n", err.Error())
	}

	c.source.Set(key, val)
}

func (c *CachingStore) Has(key []byte) bool {
	data, err := c.cache.Get(string(key))
	exists := true

	if err != nil {
		switch e := err.(type) {
		case *bigcache.EntryNotFoundError:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source
			log.Printf("[CachingStore] error while getting key from cache, error:%v\n", e.Error())
		}

		data = c.source.Get(key)
		if data == nil {
			exists = false
		} else {
			exists = true
			setErr := c.cache.Set(string(key), data)
			if setErr != nil {
				log.Printf("[CachingStore] error while setting key in cache, error: %v\n", setErr.Error())
			}
		}
	}

	return exists
}

func (c *CachingStore) Range(prefix []byte) plugin.RangeData {
	return c.source.Range(prefix)
}

func (c *CachingStore) Get(key []byte) []byte {
	data, err := c.cache.Get(string(key))

	if err != nil {
		switch e := err.(type) {
		case *bigcache.EntryNotFoundError:
			break
		default:
			// Since, there is no provision of passing error in the interface
			// we would directly access source
			log.Printf("[CachingStore] error while getting key from cache, error:%v\n", e.Error())
		}

		data = c.source.Get(key)
		if data == nil {
			return nil
		}
		setErr := c.cache.Set(string(key), data)
		if setErr != nil {
			log.Printf("[CachingStore] error while setting key in cache, error: %v\n", setErr.Error())
		}
	}
	return data
}

func (c *CachingStore) Hash() []byte {
	return c.source.Hash()
}

func (c *CachingStore) Version() int64 {
	return c.source.Version()
}

func (c *CachingStore) SaveVersion() ([]byte, int64, error) {
	return c.source.SaveVersion()
}

func (c *CachingStore) Prune() error {
	return c.source.Prune()
}
