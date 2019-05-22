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
	version int64
}

func NewMockStore() *MockStore {
	return &MockStore{
		storage: make(map[string][]byte),
		version: 0,
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
	return m.version
}

func (m *MockStore) SaveVersion() ([]byte, int64, error) {
	m.version = m.version + 1
	return nil, m.version, nil
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

	mockStore := NewMockStore()

	cachingStore, err := NewCachingStore(mockStore, defaultConfig)
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
	cachedValue, err = cachingStore.cache.Get("key1")
	require.Nil(t, err)
	assert.Equal(t, "value3", string(cachedValue), "cachingStore need to set correct value in the cache")

	cachingStore.Delete([]byte("key1"))
	storedValue = mockStore.Get([]byte("key1"))
	assert.Equal(t, true, storedValue == nil, "cachingStore need to delete value from underlying storage")
	_, err = cachingStore.cache.Get("key1")
	require.EqualError(t, err, bigcache.ErrEntryNotFound.Error())
}

func TestCachingStoreSnapshot(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true
	mockStore := NewMockStore()

	cachingStore, err := NewCachingStore(mockStore, defaultConfig)
	require.NoError(t, err)

	mockStore.Set([]byte("key1"), []byte("value1"))
	cachingStoreSnapshot := cachingStore.GetSnapshot()
	cachedValue := cachingStoreSnapshot.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingStoreSnapshot need to fetch key correctly from backing store")

	mockStore.Set([]byte("key1"), []byte("value2"))
	cachedValue = cachingStoreSnapshot.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingStoreSnapshot need to fetch key from cache and not backing store")
}
