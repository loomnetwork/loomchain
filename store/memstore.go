package store

import (
	"github.com/loomnetwork/go-loom/util"
	"strings"

	"github.com/loomnetwork/go-loom/plugin"
)

type MemStore struct {
	store map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{
		store: make(map[string][]byte),
	}
}

func (m *MemStore) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	for key, value := range m.store {
		if strings.HasPrefix(key, string(prefix)) == true {
			r := &plugin.RangeEntry{
				Key:   util.UnPrefixKey(prefix, []byte(key)),
				Value: value,
			}

			ret = append(ret, r)
		}
	}

	return ret
}

// Get returns nil iff key doesn't exist. Panics on nil key.
func (m *MemStore) Get(key []byte) []byte {
	return m.store[string(key)]
}

// Has checks if a key exists.
func (m *MemStore) Has(key []byte) bool {
	_, ok := m.store[string(key)]
	return ok
}

// Set sets the key. Panics on nil key.
func (m *MemStore) Set(key, value []byte) {
	m.store[string(key)] = value
}

// Delete deletes the key. Panics on nil key.
func (m *MemStore) Delete(key []byte) {
	delete(m.store, string(key))
}

func (m *MemStore) Hash() []byte {
	// TODO: compute some sensible hash
	return []byte("123")
}
