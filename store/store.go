package store

import (
	"github.com/loomnetwork/loom/util"
)

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
	SaveVersion() ([]byte, int64, error)
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

type prefixReader struct {
	prefix []byte
	reader KVReader
}

func (r *prefixReader) Get(key []byte) []byte {
	return r.reader.Get(util.PrefixKey(r.prefix, key))
}

func (r *prefixReader) Has(key []byte) bool {
	return r.reader.Has(util.PrefixKey(r.prefix, key))
}

func PrefixKVReader(prefix []byte, reader KVReader) KVReader {
	return &prefixReader{
		prefix: prefix,
		reader: reader,
	}
}

type prefixWriter struct {
	prefix []byte
	writer KVWriter
}

func (w *prefixWriter) Set(key, val []byte) {
	w.writer.Set(util.PrefixKey(w.prefix, key), val)
}

func (w *prefixWriter) Delete(key []byte) {
	w.writer.Delete(util.PrefixKey(w.prefix, key))
}

func PrefixKVWriter(prefix []byte, writer KVWriter) KVWriter {
	return &prefixWriter{
		prefix: prefix,
		writer: writer,
	}
}

type prefixStore struct {
	prefixReader
	prefixWriter
}

func PrefixKVStore(prefix []byte, store KVStore) KVStore {
	return &prefixStore{
		prefixReader{
			prefix: prefix,
			reader: store,
		},
		prefixWriter{
			prefix: prefix,
			writer: store,
		},
	}
}
