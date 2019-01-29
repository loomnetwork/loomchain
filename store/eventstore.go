package store

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
)

type EventStore interface {
	// Set
	SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error
	SetEventByContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIdex uint64, event []byte) error
	// Get

	// ContractID mapping
	// The plugin name is the name of the Go contract, or the address of an EVM contract.
	SetContractID(pluginName string, id uint64) error
	GetContractID(pluginName string) (uint64, error)
	// NextCotnractID generated next id in sequence
	NextContractID() uint64
}

type KVEventStore struct {
	db.DBWrapper
	sync.Mutex
}

var _ EventStore = &KVEventStore{}

func NewKVEventStore(dbWrapper db.DBWrapper) *KVEventStore {
	return &KVEventStore{DBWrapper: dbWrapper}
}

func (s *KVEventStore) SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIdex), event)
	return nil
}

func (s *KVEventStore) SetContractID(pluginName string, id uint64) error {
	// TODO: change string to bytes
	s.Set(prefixPluginName(pluginName), []byte(fmt.Sprintf("%d", id)))
	return nil
}

func (s *KVEventStore) GetContractID(pluginName string) (uint64, error) {
	// TODO: change string to bytes
	data := s.Get(prefixPluginName(pluginName))
	id, _ := strconv.ParseUint(string(data), 10, 64)
	return id, nil
}

func (s *KVEventStore) NextContractID() uint64 {
	s.Lock()
	defer s.Unlock()
	data := s.Get(prefixLastContractID())
	id, _ := strconv.ParseUint(string(data), 10, 64)
	id++
	s.Set(prefixLastContractID(), []byte(fmt.Sprintf("%d", id)))
	return id
}

func (s *KVEventStore) SetEventByContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIdex), event)
	return nil
}

type MockEventStore struct {
	*MemStore
	sync.Mutex
}

var _ EventStore = &MockEventStore{}

func NewMockEventStore(memstore *MemStore) *MockEventStore {
	return &MockEventStore{MemStore: memstore}
}

func (s *MockEventStore) SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIdex), event)
	return nil
}

func (s *MockEventStore) SetEventByContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIdex), event)
	return nil
}

func (s *MockEventStore) SetContractID(pluginName string, id uint64) error {
	// TODO: change string to bytes
	s.Set(prefixPluginName(pluginName), []byte(fmt.Sprintf("%d", id)))
	return nil
}

func (s *MockEventStore) GetContractID(pluginName string) (uint64, error) {
	// TODO: change string to bytes
	data := s.Get(prefixPluginName(pluginName))
	id, _ := strconv.ParseUint(string(data), 10, 64)
	return id, nil
}

func (s *MockEventStore) NextContractID() uint64 {
	s.Lock()
	defer s.Unlock()
	data := s.Get(prefixLastContractID())
	id, _ := strconv.ParseUint(string(data), 10, 64)
	id++
	s.Set(prefixLastContractID(), []byte(fmt.Sprintf("%d", id)))
	return id
}

func prefixBlockHightEventIndex(blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{1}, []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func prefixPluginName(pluginName string) []byte {
	return util.PrefixKey([]byte{2}, []byte(pluginName))
}

func prefixContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{3}, []byte(fmt.Sprintf("%d", contractID)), []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func prefixPluginNameTopic(pluginName string, topic string) []byte {
	return util.PrefixKey([]byte{4}, []byte(pluginName), []byte(topic))
}

func prefixLastContractID() []byte {
	return util.PrefixKey([]byte{5}, []byte("contract-id"))
}
