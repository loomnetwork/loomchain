package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestCachingStoreVersion(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true
	version := int64(1)

	mockStore := NewMockStore()

	cachingStore, err := NewVersionedCachingStore(mockStore, defaultConfig, version)

	require.NoError(t, err)

	key1 := []byte("key1")
	key2 := []byte("key2")
	key3 := []byte("key3")

	mockStore.Set(key1, []byte("value1"))
	mockStore.Set(key2, []byte("value2"))
	mockStore.Set(key3, []byte("value3"))

	snapshotv1 := cachingStore.GetSnapshot()

	// cachingStoreSnapshot will cache key1 in memory as version 1
	cachedValue := snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "cachingstore read needs to be consistent with underlying store")
	// Set data directly without update the cache, caching store should return old data
	mockStore.Set(key2, []byte("value2"))
	cachedValue = snapshotv1.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingstore need to fetch key directly from the backing store")

	cachingStore.version = 2
	cachingStore.Set(key2, []byte("newvalue2"))
	cachingStore.Set(key3, []byte("newvalue3"))
	snapshotv2 := cachingStore.GetSnapshot()
	cachedValue = snapshotv2.Get(key2)
	assert.Equal(t, "newvalue2", string(cachedValue), "snapshotv2 should not get correct value")
	cachedValue = snapshotv2.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv2 should not get correct value")

	// snapshotv1 should not get updated
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value")
	cachedValue = snapshotv1.Get(key2)
	assert.Equal(t, "value2", string(cachedValue), "snapshotv1 should get correct value")
	cachedValue = snapshotv1.Get(key3)
	assert.Equal(t, "value3", string(cachedValue), "snapshotv1 should get correct value")

	cacheSnapshot := snapshotv1.(*versionedCachingStoreSnapshot)
	cacheSnapshot.cache.Delete(key1, 1) // evict a key
	cachedValue = snapshotv1.Get(key1)  // call an evicted key
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value, fetching from underlying snapshot")

	cachingStore.version = 100
	snapshotv100 := cachingStore.GetSnapshot()
	cachedValue = snapshotv100.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv100 should get the value from cache")
	cachedValue = snapshotv100.Get(key2)
	assert.Equal(t, "newvalue2", string(cachedValue), "snapshotv100 should get the value from cache")
	cachedValue = snapshotv100.Get(key3)
	assert.Equal(t, "newvalue3", string(cachedValue), "snapshotv100 should get the value from cache")
	cacheSnapshot = snapshotv1.(*versionedCachingStoreSnapshot)
	cacheSnapshot.cache.cache.Delete(string(key1)) // evict a key table
	cachedValue = snapshotv100.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv100 should get the value from cache")
}
