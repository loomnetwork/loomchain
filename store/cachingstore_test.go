package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/allegro/bigcache"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/stretchr/testify/require"
)

type MockStore struct {
	storage map[string][]byte
}

func NewMockStore() *MockStore {
	return &MockStore{
		storage: make(map[string][]byte),
	}
}

func (m *MockStore) Get(key []byte) []byte {
	return m.storage[string(key)]
}

func (m *MockStore) Has(key []byte) bool {
	return m.storage[string(key)] != nil
}

func (m *MockStore) Set(key []byte, value []byte) {
	m.storage[string(key)] = value
}

func (m *MockStore) Delete(key []byte) {
	delete(m.storage, string(key))
}

func (m *MockStore) Range(prefix []byte) plugin.RangeData {
	return nil
}

func (m *MockStore) Hash() []byte {
	return nil
}

func (m *MockStore) Version() int64 {
	return 0
}

func (m *MockStore) SaveVersion() ([]byte, int64, error) {
	return nil, 0, nil
}

func (m *MockStore) Prune() error {
	return nil
}

func (m *MockStore) GetSnapshot() Snapshot {
	snapshotStore := make(map[string][]byte)
	for k, v := range m.storage {
		snapshotStore[k] = v
	}
	mstore := &MockStore{
		storage: snapshotStore,
	}
	return &mockStoreSnapshot{
		MockStore: mstore,
	}
}

type mockStoreSnapshot struct {
	*MockStore
}

func (s *mockStoreSnapshot) Release() {
	// noop
}

func TestCachingStore(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true
	version := int64(1)

	mockStore := NewMockStore()

	cachingStore, err := NewCachingStore(mockStore, defaultConfig, 0)
	require.NoError(t, err)

	mockStore.Set([]byte("key1"), []byte("value1"))

	cachedValue := cachingStore.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingstore read needs to be consistent with underlying store")

	// CachingStore shouldnt do anything on reads, and underlying store should serve read request
	mockStore.Set([]byte("key1"), []byte("value2"))
	cachedValue = cachingStore.Get([]byte("key1"))
	assert.Equal(t, "value2", string(cachedValue), "cachingstore need to fetch key directly from the backing store")

	cachingStore.Set([]byte("key1"), []byte("value3"))
	storedValue := mockStore.Get([]byte("key1"))
	assert.Equal(t, "value3", string(storedValue), "cachingstore need to set correct value to backing store")
	cachedValue, err = cachingStore.cache.Get([]byte("key1"), version)
	require.Nil(t, err)
	assert.Equal(t, "value3", string(cachedValue), "cachingStore need to set correct value in the cache")

	cachingStore.Delete([]byte("key1"))
	storedValue = mockStore.Get([]byte("key1"))
	assert.Equal(t, true, storedValue == nil, "cachingStore need to delete value from underlying storage")
	cachedValue, err = cachingStore.cache.Get([]byte("key1"), version)
	require.EqualError(t, err, bigcache.ErrEntryNotFound.Error())
}

func TestCachingStoreSnapshot(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true
	version := int64(1)
	mockStore := NewMockStore()

	cachingStore, err := NewCachingStore(mockStore, defaultConfig, version)
	require.NoError(t, err)

	mockStore.Set([]byte("key1"), []byte("value1"))
	cachingStoreSnapshot := cachingStore.GetSnapshot()
	cachedValue := cachingStoreSnapshot.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingStoreSnapshot need to fetch key correctly from backing store")

	mockStore.Set([]byte("key1"), []byte("value2"))
	cachedValue = cachingStoreSnapshot.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingStoreSnapshot need to fetch key from cache and not backing store")
}

func TestCachingStoreVersion(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true
	version := int64(1)

	mockStore := NewMockStore()

	cachingStore, err := NewCachingStore(mockStore, defaultConfig, version)

	require.NoError(t, err)

	mockStore.Set([]byte("key1"), []byte("value1"))
	mockStore.Set([]byte("key2"), []byte("value2"))
	mockStore.Set([]byte("key3"), []byte("value3"))

	snapshotv1 := cachingStore.GetSnapshot()

	// cachingStoreSnapshot will cache key1 in memory as version 1
	cachedValue := snapshotv1.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingstore read needs to be consistent with underlying store")
	// Set data directly without update the cache, caching store should return old data
	mockStore.Set([]byte("key1"), []byte("value2"))
	cachedValue = snapshotv1.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingstore need to fetch key directly from the backing store")

	cachingStore.version = 2
	cachingStore.Set([]byte("key2"), []byte("newvalue2"))
	cachingStore.Set([]byte("key3"), []byte("newvalue3"))
	snapshotv2 := cachingStore.GetSnapshot()
	cachedValue = snapshotv2.Get([]byte("key2"))
	assert.Equal(t, "newvalue2", string(cachedValue), "snapshotv2 should not get correct value")
	cachedValue = snapshotv2.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "snapshotv2 should not get correct value")

	// snapshotv1 should not get updated
	cachedValue = snapshotv1.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value")
	cachedValue = snapshotv1.Get([]byte("key2"))
	assert.Equal(t, "value2", string(cachedValue), "snapshotv1 should get correct value")
	cachedValue = snapshotv1.Get([]byte("key3"))
	assert.Equal(t, "value3", string(cachedValue), "snapshotv1 should get correct value")

	cacheSnapshot := snapshotv1.(*CachingStoreSnapshot)
	cacheSnapshot.cache.Delete([]byte("key1"), 1) // evict a key
	cachedValue = snapshotv1.Get([]byte("key1"))  // call an evicted key
	assert.Equal(t, "value1", string(cachedValue), "snapshotv1 should get correct value, fetching from underlying snapshot")

	cachingStore.version = 100
	snapshotv100 := cachingStore.GetSnapshot()
	cachedValue = snapshotv100.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "snapshotv100 should get the value from cache")
}
