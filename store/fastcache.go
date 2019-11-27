package store

import (
	"github.com/VictoriaMetrics/fastcache"
	loom "github.com/loomnetwork/go-loom"
)

type versionedFastCache struct {
	versionedBaseCache
	cache *fastcache.Cache
}

func newVersionedFastCache(config *CachingStoreConfig, cacheLogger *loom.Logger) (*versionedFastCache, error) {
	versionedCache := &versionedFastCache{
		versionedBaseCache: versionedBaseCache{
			cacheLogger: cacheLogger,
			keyTable:    map[string]KeyVersionTable{},
		},
	}

	cache := fastcache.New(config.MaxKeys * config.MaxSizeOfValueInBytes)
	versionedCache.cache = cache
	return versionedCache, nil
}

func (c *versionedFastCache) Delete(key []byte, version int64) error {
	versionedKey := []byte(versionedKey(string(key), version))
	// delete data in cache if it does exist
	c.cache.Del(versionedKey)
	// add key to inidicate that this is the latest version but
	// the data has been deleted
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedFastCache) Set(key, val []byte, version int64) error {
	versionedKey := []byte(versionedKey(string(key), version))
	c.cache.Set(versionedKey, val)
	c.addKeyVersion(key, version)
	return nil
}

func (c *versionedFastCache) Get(key []byte, version int64) ([]byte, error) {
	latestVersion := c.getKeyVersion(key, version)
	versionedKey := []byte(versionedKey(string(key), latestVersion))
	return c.cache.Get(nil, versionedKey), nil
}
