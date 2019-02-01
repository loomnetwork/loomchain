package store

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
)

type EventStore interface {
	// SetEvent stores event into storage
	// contractID, blockHeight, and eventIndex are required for makeing a unique keys
	SetEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error
	// FilterEvents filters events that match the given filter
	FilterEvents(filter *types.EventFilter) ([]*types.EventData, error)
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

func (s *KVEventStore) SetEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error {
	data, err := proto.Marshal(eventData)
	if err != nil {
		return err
	}
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIndex), data)
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIndex), data)
	return nil
}

func (s *KVEventStore) FilterEvents(filter *types.EventFilter) ([]*types.EventData, error) {
	// TODO improve it
	var events []*types.EventData
	if filter.Contract != "" {
		contractID, _ := s.GetContractID(filter.Contract)
		if contractID == 0 {
			return nil, fmt.Errorf("contract not found: %s", filter.Contract)
		}
		// Interator uses [start, end) so make sure we increase end inclusively
		start := prefixContractIDBlockHight(contractID, filter.FromBlock)
		end := prefixContractIDBlockHight(contractID, filter.ToBlock+1)
		itr := s.Iterator(start, end)
		defer itr.Close()
		for ; itr.Valid(); itr.Next() {
			var ed types.EventData
			if err := proto.Unmarshal(itr.Value(), &ed); err != nil {
				return nil, err
			}
			events = append(events, &ed)
		}
	} else {
		// Interator uses [start, end) so make sure we increase end inclusively
		start := prefixBlockHight(filter.FromBlock)
		end := prefixBlockHight(filter.ToBlock + 1)
		itr := s.Iterator(start, end)
		defer itr.Close()
		for ; itr.Valid(); itr.Next() {
			var ed types.EventData
			if err := proto.Unmarshal(itr.Value(), &ed); err != nil {
				return nil, err
			}
			events = append(events, &ed)
		}
	}

	return events, nil
}

func (s *KVEventStore) SetContractID(pluginName string, id uint64) error {
	s.Set(prefixPluginName(pluginName), uint64ToBytes(id))
	return nil
}

func (s *KVEventStore) GetContractID(pluginName string) (uint64, error) {
	data := s.Get(prefixPluginName(pluginName))
	return bytesToUint64(data), nil
}

func (s *KVEventStore) NextContractID() uint64 {
	s.Lock()
	defer s.Unlock()
	data := s.Get(prefixLastContractID())
	id := bytesToUint64(data)
	id++
	s.Set(prefixLastContractID(), uint64ToBytes(id))
	return id
}

type MockEventStore struct {
	*MemStore
	sync.Mutex
}

var _ EventStore = &MockEventStore{}

func NewMockEventStore(memstore *MemStore) *MockEventStore {
	return &MockEventStore{MemStore: memstore}
}

func (s *MockEventStore) SetEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error {
	data, err := proto.Marshal(eventData)
	if err != nil {
		return err
	}
	s.Set(prefixBlockHightEventIndex(blockHeight, eventIndex), data)
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIndex), data)
	return nil
}

func (s *MockEventStore) FilterEvents(filter *types.EventFilter) ([]*types.EventData, error) {
	// events must be returned as ordered list
	var events []*types.EventData
	if filter.Contract != "" {
		contractID, _ := s.GetContractID(filter.Contract)
		if contractID == 0 {
			return nil, fmt.Errorf("contract id not found: %s", filter.Contract)
		}
		for i := filter.FromBlock; i <= filter.ToBlock; i++ {
			rangeData := s.Range(prefixContractIDBlockHight(contractID, i))
			for _, entry := range rangeData {
				var ed types.EventData
				if err := proto.Unmarshal(entry.Value, &ed); err != nil {
					return nil, err
				}
				events = append(events, &ed)
			}
		}
	} else {
		for i := filter.FromBlock; i <= filter.ToBlock; i++ {
			rangeData := s.Range(prefixBlockHight(i))
			for _, entry := range rangeData {
				var ed types.EventData
				if err := proto.Unmarshal(entry.Value, &ed); err != nil {
					return nil, err
				}
				events = append(events, &ed)
			}
		}
	}

	// sort the events by block height and event index
	sort.Slice(events, func(i, j int) bool {
		if events[i].BlockHeight == events[j].BlockHeight {
			return events[i].TransactionIndex < events[j].TransactionIndex
		}
		return events[i].BlockHeight < events[j].BlockHeight
	})

	return events, nil
}

func (s *MockEventStore) SetContractID(pluginName string, id uint64) error {
	s.Set(prefixPluginName(pluginName), uint64ToBytes(id))
	return nil
}

func (s *MockEventStore) GetContractID(pluginName string) (uint64, error) {
	data := s.Get(prefixPluginName(pluginName))
	return bytesToUint64(data), nil
}

func (s *MockEventStore) NextContractID() uint64 {
	s.Lock()
	defer s.Unlock()
	data := s.Get(prefixLastContractID())
	id := bytesToUint64(data)
	id++
	s.Set(prefixLastContractID(), uint64ToBytes(id))
	return id
}

func prefixBlockHightEventIndex(blockHeight uint64, eventIndex uint16) []byte {
	return util.PrefixKey([]byte{1}, uint64ToBytes(blockHeight), uint16ToBytes(eventIndex))
}

func prefixBlockHight(blockHeight uint64) []byte {
	return util.PrefixKey([]byte{1}, uint64ToBytes(blockHeight))
}

func prefixPluginName(pluginName string) []byte {
	return util.PrefixKey([]byte{2}, []byte(pluginName))
}

func prefixContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIndex uint16) []byte {
	return util.PrefixKey([]byte{3}, uint64ToBytes(contractID), uint64ToBytes(blockHeight), uint16ToBytes(eventIndex))
}

func prefixContractIDBlockHight(contractID uint64, blockHeight uint64) []byte {
	return util.PrefixKey([]byte{3}, uint64ToBytes(contractID), uint64ToBytes(blockHeight))
}

func prefixPluginNameTopic(pluginName string, topic string) []byte {
	return util.PrefixKey([]byte{4}, []byte(pluginName), []byte(topic))
}

func prefixLastContractID() []byte {
	return util.PrefixKey([]byte{5}, []byte("contract-id"))
}

func uint64ToBytes(n uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, n)
	return buf
}

func uint16ToBytes(n uint16) []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, n)
	return buf
}

func bytesToUint64(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}
