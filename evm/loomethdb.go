// +build evm

package evm

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"sync"
)

// implements ethdb.Database
type LoomEthdb struct {
	ctx   context.Context
	state store.KVStore
	lock  sync.RWMutex
}

func NewLoomEthdb(_state loomchain.State) *LoomEthdb {
	p := new(LoomEthdb)
	p.ctx = _state.Context()
	p.state = store.PrefixKVStore(vmPrefix, _state)
	return p
}

func (s *LoomEthdb) Put(key []byte, value []byte) error {
	s.state.Set(key, value)
	return nil
}

func (s *LoomEthdb) Get(key []byte) ([]byte, error) {
	return s.state.Get(key), nil
}

func (s *LoomEthdb) Has(key []byte) (bool, error) {
	return s.state.Has(key), nil
}

func (s *LoomEthdb) Delete(key []byte) error {
	s.state.Delete(key)
	return nil
}

func (s *LoomEthdb) Close() {
}

func (s *LoomEthdb) NewBatch() ethdb.Batch {
	return &memBatch{db: s}
}

type kv struct{ k, v []byte }

// implements ethdb.Batch
// https://github.com/ethereum/go-ethereum/blob/master/ethdb/memory_database.go#L101
type memBatch struct {
	db     *LoomEthdb
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *memBatch) Delete(key []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil})
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.v == nil {
			b.db.Delete(kv.k)
			continue
		}
		b.db.Put(kv.k, kv.v)
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
