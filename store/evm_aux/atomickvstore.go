package evmaux

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

type atomicKVStore struct {
	db    dbm.DB
	cache map[string][]byte
}

func NewAtomicKVStore(db dbm.DB) *atomicKVStore {
	return &atomicKVStore{
		db:    db,
		cache: make(map[string][]byte),
	}
}

func (a *atomicKVStore) Delete(key []byte) {
	a.cache[string(key)] = nil
}

func (a *atomicKVStore) Set(key, val []byte) {
	a.cache[string(key)] = val
}

func (a *atomicKVStore) Has(key []byte) bool {
	val, ok := a.cache[string(key)]
	if ok {
		return val != nil
	}
	return a.db.Has(key)
}

func (a *atomicKVStore) Get(key []byte) []byte {
	if _, ok := a.cache[string(key)]; ok {
		return a.cache[string(key)]
	}

	return a.db.Get(key)
}

func (a *atomicKVStore) Commit() {
	batch := a.db.NewBatch()
	for key, value := range a.cache {
		if value != nil {
			batch.Set([]byte(key), value)
		} else {
			batch.Delete([]byte(key))
		}
	}
	batch.WriteSync()
	a.Rollback()
}

func (a *atomicKVStore) Rollback() {
	a.cache = make(map[string][]byte)
}
