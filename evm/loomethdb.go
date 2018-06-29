// +build evm

package evm

import (
	"context"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

// implements ethdb.Database
type LoomEthdb struct {
	ctx   context.Context
	state store.KVStore
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
	newBatch := new(batch)
	newBatch.parentStore = s
	newBatch.Reset()
	return newBatch
}

// implements ethdb.batch
type kvPair struct {
	key   []byte
	value []byte
}

type batch struct {
	cache       []kvPair
	parentStore *LoomEthdb
}

func (b *batch) Put(key []byte, value []byte) error {
	b.cache = append(b.cache, kvPair{
		key:   key,
		value: value,
	})
	return nil
}

func (b *batch) ValueSize() int {
	return len(b.cache)
}

func (b *batch) Write() error {
	for _, kv := range b.cache {
		b.parentStore.Put(kv.key, kv.value)
	}
	return nil
}

func (b *batch) Reset() {
	b.cache = make([]kvPair, 0)
}
