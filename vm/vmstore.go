package vm

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/loom"
)

// implentns ethdb.Database
type evmStore struct {
	state loom.State
}

func NewEvmStore(_state loom.State) (*evmStore){
	p := new(evmStore)
	p.state = _state
	return p
}

func (s *evmStore) Put(key []byte, value []byte) error {
	s.state.Set(key,value)
	return nil
}

func (s *evmStore) Get(key []byte) ([]byte, error) {
	return s.state.Get(key), nil
}

func (s *evmStore) Has(key []byte) (bool, error) {
	return s.state.Has(key), nil
}

func (s *evmStore) Delete(key []byte) (error) {
	s.state.Delete(key)
	return nil
}

func (s *evmStore) Close()  {
}

func (s *evmStore) NewBatch() (ethdb.Batch) {
	newBatch := new(batch)
	newBatch.parentStore = s
	newBatch.cache = make(map[string][]byte)
	return newBatch
}

// implements ethdb.batch
type batch struct {
	cache map[string][]byte
	parentStore* evmStore
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
		b.parentStore.Put(common.Hex2Bytes(k),v)
	}
	return nil
}

func (b *batch) Reset() {
	b.cache = make(map[string][]byte)
}

