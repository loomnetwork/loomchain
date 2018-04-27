// +build evm

package vm

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

var vmPrefix = []byte("vm")

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
	newBatch.cache = make(map[string][]byte)
	return newBatch
}

// implements ethdb.batch
type batch struct {
	cache       map[string][]byte
	parentStore *LoomEthdb
}

func (b *batch) Put(key []byte, value []byte) error {
	keyStr := common.Bytes2Hex(key)
	b.cache[keyStr] = value
	return nil
}

func (b *batch) ValueSize() int {
	return len(b.cache)
}

func (b *batch) Write() error {
	for k, v := range b.cache {
		b.parentStore.Put(common.Hex2Bytes(k), v)
	}
	return nil
}

func (b *batch) Reset() {
	b.cache = make(map[string][]byte)
}
