package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachingStoreVersion(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true

	mockStore := NewMockStore()

	versionedStore, err := NewVersionedCachingStore(mockStore, defaultConfig, mockStore.Version())
	cachingStore := versionedStore.(*versionedCachingStore)

	require.NoError(t, err)

	key1 := []byte("key1")
	key2 := []byte("key2")
	key3 := []byte("key3")

	mockStore.Set(key1, []byte("value1"))
	mockStore.Set(key2, []byte("value2"))
	mockStore.Set(key3, []byte("value3"))

	snapshotv0 := cachingStore.GetSnapshot()

	// cachingStoreSnapshot will cache key1 in memory as version 0
	cachedValue := snapshotv0.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "cachingstore read needs to be consistent with underlying store")
	// Set data directly without update the cache, caching store should return old data
	mockStore.Set(key2, []byte("value2"))
	cachedValue = snapshotv0.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingstore need to fetch key directly from the backing store")

	// save to bump up version
	_, version, _ := cachingStore.SaveVersion()
	assert.Equal(t, int64(1), version, "version must be updated to 1")
	// save data into version 1
	cachingStore.Set(key2, []byte("newvalue2"))
	cachingStore.Set(key3, []byte("newvalue3"))
	snapshotv1 := cachingStore.GetSnapshot()
	cachedValue = snapshotv1.Get(key2)
	assert.Equal(t, "newvalue2", string(cachedValue), "snapshotv1 should get correct value")
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value")

	// snapshotv0 should not get updated
	cachedValue = snapshotv0.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv0 should get correct value")
	cachedValue = snapshotv0.Get(key2)
	assert.Equal(t, "value2", string(cachedValue), "snapshotv0 should get correct value")
	cachedValue = snapshotv0.Get(key3)
	assert.Equal(t, "value3", string(cachedValue), "snapshotv0 should get correct value")

	cacheSnapshot := snapshotv0.(*versionedCachingStoreSnapshot)
	cacheSnapshot.cache.Delete(key1, 1) // evict a key
	cachedValue = snapshotv0.Get(key1)  // call an evicted key
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value, fetching from underlying snapshot")

	// save to bump up version
	_, version, _ = cachingStore.SaveVersion()
	assert.Equal(t, int64(2), version, "version must be updated to 2")
	snapshotv2 := cachingStore.GetSnapshot()
	cachedValue = snapshotv2.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv2 should get the value from cache")
	cachedValue = snapshotv2.Get(key2)
	assert.Equal(t, "newvalue2", string(cachedValue), "snapshotv2 should get the value from cache")
	cachedValue = snapshotv2.Get(key3)
	assert.Equal(t, "newvalue3", string(cachedValue), "snapshotv2 should get the value from cache")

	// evict data from key table
	cacheSnapshot = snapshotv1.(*versionedCachingStoreSnapshot)
	cacheSnapshot.cache.cache.Delete(string(key1)) // evict a key table
	cachedValue = snapshotv2.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshotv2 should get the value from cache")
}
