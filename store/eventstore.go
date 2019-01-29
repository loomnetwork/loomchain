package store

import (
	"fmt"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
)

type EventStore interface {
	// Set
	SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error
	SetEventByPluginName(pluginName string, event []byte) error
	SetEventByContractIDBlockHightEventIndex(contractID []byte, blockHeight uint64, eventIdex uint64, event []byte) error
	// TODO: Get
}

type KVEventStore struct {
	db.DBWrapper
}

var _ EventStore = &KVEventStore{}

func NewKVEventStore(dbWrapper db.DBWrapper) *KVEventStore {
	return &KVEventStore{DBWrapper: dbWrapper}
}

func (s *KVEventStore) SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIdex), event)
	return nil
}

func (s *KVEventStore) SetEventByPluginName(pluginName string, event []byte) error {
	s.Set(prefixPluginName(pluginName), event)
	return nil
}

func (s *KVEventStore) SetEventByContractIDBlockHightEventIndex(contractID []byte, blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIdex), event)
	return nil
}

type MockEventStore struct {
	*MemStore
}

var _ EventStore = &MockEventStore{}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{MemStore: NewMemStore()}
}

func (s *MockEventStore) SetEventByBlockHightEventIndex(blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIdex), event)
	return nil
}

func (s *MockEventStore) SetEventByPluginName(pluginName string, event []byte) error {
	s.Set(prefixPluginName(pluginName), event)
	return nil
}

func (s *MockEventStore) SetEventByContractIDBlockHightEventIndex(contractID []byte, blockHeight uint64, eventIdex uint64, event []byte) error {
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIdex), event)
	return nil
}

func prefixBlockHightEventIndex(blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{1}, []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func prefixPluginName(pluginName string) []byte {
	return util.PrefixKey([]byte{2}, []byte(pluginName))
}

func prefixContractIDBlockHightEventIndex(contractID []byte, blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{3}, contractID, []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func prefixPluginNameTopic(pluginName string, topic string) []byte {
	return util.PrefixKey([]byte{4}, []byte(pluginName), []byte(topic))
}
