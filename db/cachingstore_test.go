package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tendermint/libs/db"
)

type MockDB struct {
	storage map[string][]byte
}

func NewMockDB() *MockDB {
	return &MockDB{
		storage: make(map[string][]byte),
	}
}

func (m *MockDB) Get(key []byte) []byte {
	return m.storage[string(key)]
}

func (m *MockDB) Has(key []byte) bool {
	return m.storage[string(key)] != nil
}

func (m *MockDB) SetSync(key []byte, value []byte) {
	m.storage[string(key)] = value
}

func (m *MockDB) Set(key []byte, value []byte) {
	m.storage[string(key)] = value
}

func (m *MockDB) Delete(key []byte) {
	delete(m.storage, string(key))
}

func (m *MockDB) DeleteSync(key []byte) {
	delete(m.storage, string(key))
}

func (m *MockDB) BatchDelete(key []byte) {

}

func (m *MockDB) BatchSet(key []byte, value []byte) {

}

func (m *MockDB) FlushBatch() {

}

func (m *MockDB) Iterator(start, end []byte) dbm.Iterator {
	return nil
}

func (m *MockDB) ReverseIterator(start, end []byte) dbm.Iterator {
	return nil

}

func (m *MockDB) Close() {

}

func (m *MockDB) NewBatch() dbm.Batch {
	return nil
}

func (m *MockDB) Print() {

}

func (m *MockDB) Stats() map[string]string {
	return nil
}

func (m *MockDB) Compact() error {
	return nil
}

func TestCachingDB(t *testing.T) {
	defaultConfig := DefaultCachingDBConfig()
	defaultConfig.CachingEnabled = true

	mockDB := NewMockDB()

	cachingDB, err := NewCachingDB(mockDB, defaultConfig)
	require.NoError(t, err)

	mockDB.Set([]byte("key1"), []byte("value1"))

	cachedValue := cachingDB.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingDB read needs to be consistent with underlying store")

	// CachingDB shouldnt do anything on reads, and underlying store should serve read request
	mockDB.Set([]byte("key1"), []byte("value2"))
	cachedValue = cachingDB.Get([]byte("key1"))
	assert.Equal(t, "value2", string(cachedValue), "cachingDB need to fetch key directly from the backing store")

	cachingDB.Set([]byte("key1"), []byte("value3"))
	storedValue := mockDB.Get([]byte("key1"))
	assert.Equal(t, "value3", string(storedValue), "cachingDB need to set correct value to backing store")
	cachedValue, err = cachingDB.cache.Get("key1")
	require.Nil(t, err)
	assert.Equal(t, "value3", string(cachedValue), "cachingDB need to set correct value in the cache")

	cachingDB.Delete([]byte("key1"))
	storedValue = mockDB.Get([]byte("key1"))
	assert.Equal(t, true, storedValue == nil, "cachingDB need to delete value from underlying storage")
	cachedValue, err = cachingDB.cache.Get("key1")
	require.EqualError(t, err, fmt.Sprintf("Entry %q not found", "key1"))
}
