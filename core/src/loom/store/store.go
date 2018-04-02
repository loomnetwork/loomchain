package store

type KVReader interface {
	// Get returns nil iff key doesn't exist. Panics on nil key.
	Get(key []byte) []byte

	// Has checks if a key exists.
	Has(key []byte) bool
}

type KVWriter interface {
	// Set sets the key. Panics on nil key.
	Set(key, value []byte)

	// Delete deletes the key. Panics on nil key.
	Delete(key []byte)
}

type KVStore interface {
	KVReader
	KVWriter
}

type KVStoreTx interface {
	KVStore
	Commit()
	Rollback()
}

type AtomicKVStore interface {
	KVStore
	BeginTx() KVStoreTx
}

type VersionedKVStore interface {
	KVStore
	Hash() []byte
	Version() int64
	SaveVersion() (int64, error)
}

type cacheItem struct {
	Value   []byte
	Deleted bool
}

// cacheTx is a simple write-back cache
type cacheTx struct {
	store KVStore
	cache map[string]cacheItem
}

func newCacheTx(store KVStore) *cacheTx {
	c := &cacheTx{
		store: store,
	}
	c.Rollback()
	return c
}

func (c *cacheTx) setCache(key, val []byte, deleted bool) {
	c.cache[string(key)] = cacheItem{
		Value:   val,
		Deleted: deleted,
	}
}

func (c *cacheTx) Delete(key []byte) {
	c.setCache(key, nil, true)
}

func (c *cacheTx) Set(key, val []byte) {
	c.setCache(key, val, false)
}

func (c *cacheTx) Has(key []byte) bool {
	if item, ok := c.cache[string(key)]; ok {
		return !item.Deleted
	}

	return c.store.Has(key)
}

func (c *cacheTx) Get(key []byte) []byte {
	if item, ok := c.cache[string(key)]; ok {
		return item.Value
	}

	return c.store.Get(key)
}

func (c *cacheTx) Commit() {
	for skey, item := range c.cache {
		key := []byte(skey)
		if item.Deleted {
			c.store.Delete(key)
		} else {
			c.store.Set(key, item.Value)
		}
	}
}

func (c *cacheTx) Rollback() {
	c.cache = make(map[string]cacheItem)
}

type atomicWrapStore struct {
	KVStore
}

func (a *atomicWrapStore) BeginTx() KVStoreTx {
	return newCacheTx(a)
}

func WrapAtomic(store KVStore) AtomicKVStore {
	return &atomicWrapStore{
		KVStore: store,
	}
}
