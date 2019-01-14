package db

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tendermint/libs/db"

	"github.com/allegro/bigcache"
)

const DELETE_ACTION = "delete"
const SET_ACTION = "set"

type Action struct {
	Id    string
	Key   []byte
	Value []byte
}

type MockDB struct {
	storage      map[string][]byte
	batchActions []Action
}

func NewMockDB() *MockDB {
	return &MockDB{
		storage:      make(map[string][]byte),
		batchActions: make([]Action, 0, 1),
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
	m.batchActions = append(m.batchActions, Action{Id: DELETE_ACTION, Key: key})
}

func (m *MockDB) BatchSet(key []byte, value []byte) {
	m.batchActions = append(m.batchActions, Action{Id: SET_ACTION, Key: key, Value: value})
}

func (m *MockDB) FlushBatch() {
	for _, action := range m.batchActions {
		switch action.Id {
		case DELETE_ACTION:
			delete(m.storage, string(action.Key))
			break
		case SET_ACTION:
			m.storage[string(action.Key)] = action.Value
			break
		default:
			panic("invalid action")
		}
	}
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

	mockDB.Set([]byte("key1"), []byte("value2"))

	cachedValue = cachingDB.Get([]byte("key1"))
	assert.Equal(t, "value1", string(cachedValue), "cachingDB need to fetch key from the cache")

	cachingDB.Set([]byte("key1"), []byte("value3"))
	storedValue := mockDB.Get([]byte("key1"))
	assert.Equal(t, "value2", string(storedValue), "cachingDB need not to flush write immediately")
	cachedValue, err = cachingDB.cache.Get("key1")
	require.Nil(t, err)
	assert.Equal(t, "value3", string(cachedValue), "cachingDB need to set correct value in the cache")

	mockDB.FlushBatch()
	storedValue = mockDB.Get([]byte("key1"))
	assert.Equal(t, "value3", string(storedValue), "changes should be reflected in underlying db")

	cachingDB.Delete([]byte("key1"))
	storedValue = mockDB.Get([]byte("key1"))
	assert.Equal(t, true, storedValue != nil, "cachingDB need not to flush delete immediately")
	cachedValue, err = cachingDB.cache.Get("key1")
	require.EqualError(t, err, bigcache.ErrEntryNotFound.Error())

	mockDB.FlushBatch()
	storedValue = mockDB.Get([]byte("key1"))
	assert.Equal(t, true, storedValue == nil, "changes should be reflected in underlying db")

}
