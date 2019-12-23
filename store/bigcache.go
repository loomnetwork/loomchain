package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/loomnetwork/go-loom"
)

type versionedBaseCache struct {
	cacheLogger   *loom.Logger
	keyTableMutex sync.RWMutex
	keyTable      map[string]KeyVersionTable
}

func convertToBigCacheConfig(config CachingStoreConfig, logger *loom.Logger) (*bigcache.Config, error) {
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

func (c *versionedBaseCache) onRemove(key string, entry []byte) {
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

// getKeyVersion returns the latest version number (limited by version argument) of a particular key
func (c *versionedBaseCache) getKeyVersion(key []byte, version int64) int64 {
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
func (c *versionedBaseCache) addKeyVersion(key []byte, version int64) {
	c.keyTableMutex.Lock()
	defer c.keyTableMutex.Unlock()
	kvTable, exist := c.keyTable[string(key)]
	if !exist {
		kvTable = KeyVersionTable{}
	}
	kvTable[version] = true
	c.keyTable[string(key)] = kvTable
}

type versionedBigCache struct {
	versionedBaseCache
	cache *bigcache.BigCache
}

func newVersionedBigCache(config CachingStoreConfig, cacheLogger *loom.Logger) (*versionedBigCache, error) {
	bigcacheConfig, err := convertToBigCacheConfig(config, cacheLogger)
	if err != nil {
		return nil, err
	}
	versionedCache := &versionedBigCache{
		versionedBaseCache: versionedBaseCache{
			cacheLogger: cacheLogger,
			keyTable:    map[string]KeyVersionTable{},
		},
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

func (c *versionedBigCache) Delete(key []byte, version int64) error {
	versionedKey := versionedKey(string(key), version)
	// delete data in cache if it does exist
	c.cache.Delete(versionedKey)
	// add key to inidicate that this is the latest version but
	// the data has been deleted
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedBigCache) Set(key, val []byte, version int64) error {
	versionedKey := versionedKey(string(key), version)
	err := c.cache.Set(versionedKey, val)
	if err != nil {
		return err
	}
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedBigCache) Get(key []byte, version int64) ([]byte, error) {
	latestVersion := c.getKeyVersion(key, version)
	versionedKey := versionedKey(string(key), latestVersion)
	return c.cache.Get(versionedKey)
}
