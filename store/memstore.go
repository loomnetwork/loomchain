package store

type MemStore struct {
	store map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{
		store: make(map[string][]byte),
	}
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
