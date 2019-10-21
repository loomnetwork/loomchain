package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
)

// KVReader interface for reading data out of a store
type KVReader interface {
	// Get returns nil iff key doesn't exist. Panics on nil key.
	Get(key []byte) []byte

	// Range returns a range of keys
	Range(prefix []byte) plugin.RangeData

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

type Snapshot interface {
	KVReader
	Release()
}

type VersionedKVStore interface {
	KVStore
	Hash() []byte
	Version() int64
	SaveVersion() ([]byte, int64, error)
	// Delete old version of the store
	Prune() error
	GetSnapshot(version int64) Snapshot
	VersionExists(int64) bool
	RetrieveVersion(int64) (VersionedKVStore, error)
}

type cacheItem struct {
	Value   []byte
	Deleted bool
}

type txAction int

const (
	txSet txAction = iota
	txDelete
)

type tempTx struct {
	Action     txAction
	Key, Value []byte
}

// cacheTx is a simple write-back cache
type cacheTx struct {
	store KVStore
	cache map[string]cacheItem
	// tmpTxs preserves the order of set and delete actions
	tmpTxs []tempTx
}

func newCacheTx(store KVStore) *cacheTx {
	c := &cacheTx{
		store: store,
	}
	c.Rollback()
	return c
}

func (c *cacheTx) addAction(action txAction, key, value []byte) {
	c.tmpTxs = append(c.tmpTxs, tempTx{
		Action: action,
		Key:    key,
		Value:  value,
	})
}

func (c *cacheTx) setCache(key, val []byte, deleted bool) {
	c.cache[string(key)] = cacheItem{
		Value:   val,
		Deleted: deleted,
	}
}

func (c *cacheTx) Delete(key []byte) {
	c.addAction(txDelete, key, nil)
	c.setCache(key, nil, true)
}

func (c *cacheTx) Set(key, val []byte) {
	c.addAction(txSet, key, val)
	c.setCache(key, val, false)
}

func (c *cacheTx) Range(prefix []byte) plugin.RangeData {
	//TODO cache ranges???
	return c.store.Range(prefix)
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
	for _, tx := range c.tmpTxs {
		if tx.Action == txSet {
			c.store.Set(tx.Key, tx.Value)
		} else if tx.Action == txDelete {
			c.store.Delete(tx.Key)
		} else {
			panic("invalid cacheTx action type")
		}
	}
}

func (c *cacheTx) Rollback() {
	c.tmpTxs = make([]tempTx, 0)
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

func (r *prefixReader) Range(prefix []byte) plugin.RangeData {
	return r.reader.Range(util.PrefixKey(r.prefix, prefix))
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
