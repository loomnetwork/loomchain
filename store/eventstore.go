package store

import (
	"encoding/binary"
	"sort"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	blockHeightKeyPrefix           byte = 1
	pluginNameKeyPrefix            byte = 2
	contractIDBlockHeightKeyPrefix byte = 3
	pluginNameTopicKeyPrefix       byte = 4
	lastContractIDKeyPrefix             = 5
)

type EventFilter struct {
	FromBlock uint64
	ToBlock   uint64
	Contract  string
}

type EventStore interface {
	// SaveEvent stores event into storage
	// contractID, blockHeight, and eventIndex are required for makeing a unique keys
	SaveEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error
	// FilterEvents filters events that match the given filter
	FilterEvents(filter EventFilter) ([]*types.EventData, error)
	// ContractID mapping
	GetContractID(pluginName string) uint64
}

type KVEventStore struct {
	dbm.DB
	sync.Mutex
}

var _ EventStore = &KVEventStore{}

func NewKVEventStore(db dbm.DB) *KVEventStore {
	return &KVEventStore{DB: db}
}

func (s *KVEventStore) SaveEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error {
	data, err := proto.Marshal(eventData)
	if err != nil {
		return err
	}
	s.Set(prefixBlockHeightEventIndex(blockHeight, eventIndex), data)
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIndex), data)
	return nil
}

func (s *KVEventStore) FilterEvents(filter EventFilter) ([]*types.EventData, error) {
	var events []*types.EventData
	if filter.Contract != "" {
		contractID := s.GetContractID(filter.Contract)
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
		start := prefixBlockHeightEventIndex(filter.FromBlock, 0)
		end := prefixBlockHeightEventIndex(filter.ToBlock+1, 0)
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

func (s *KVEventStore) GetContractID(pluginName string) uint64 {
	data := s.Get(prefixPluginName(pluginName))
	id := bytesToUint64(data)
	if id == 0 {
		id = s.nextContractID()
	}
	s.Set(prefixPluginName(pluginName), uint64ToBytes(id))
	return id
}

func (s *KVEventStore) nextContractID() uint64 {
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

func (s *MockEventStore) SaveEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error {
	data, err := proto.Marshal(eventData)
	if err != nil {
		return err
	}
	s.Set(prefixBlockHeightEventIndex(blockHeight, eventIndex), data)
	s.Set(prefixContractIDBlockHightEventIndex(contractID, blockHeight, eventIndex), data)
	return nil
}

func (s *MockEventStore) FilterEvents(filter EventFilter) ([]*types.EventData, error) {
	// events must be returned as ordered list
	var events []*types.EventData
	if filter.Contract != "" {
		contractID := s.GetContractID(filter.Contract)
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
			rangeData := s.Range(prefixBlockHeight(i))
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

func (s *MockEventStore) GetContractID(pluginName string) uint64 {
	data := s.Get(prefixPluginName(pluginName))
	id := bytesToUint64(data)
	if id == 0 {
		id = s.nextContractID()
	}
	s.Set(prefixPluginName(pluginName), uint64ToBytes(id))
	return id
}

func (s *MockEventStore) nextContractID() uint64 {
	s.Lock()
	defer s.Unlock()
	data := s.Get(prefixLastContractID())
	id := bytesToUint64(data)
	id++
	s.Set(prefixLastContractID(), uint64ToBytes(id))
	return id
}

func prefixBlockHeightEventIndex(blockHeight uint64, eventIndex uint16) []byte {
	return util.PrefixKey([]byte{blockHeightKeyPrefix}, uint64ToBytes(blockHeight), uint16ToBytes(eventIndex))
}

func prefixBlockHeight(blockHeight uint64) []byte {
	return util.PrefixKey([]byte{pluginNameKeyPrefix}, uint64ToBytes(blockHeight))
}

func prefixPluginName(pluginName string) []byte {
	return util.PrefixKey([]byte{pluginNameKeyPrefix}, []byte(pluginName))
}

func prefixContractIDBlockHightEventIndex(contractID uint64, blockHeight uint64, eventIndex uint16) []byte {
	return util.PrefixKey([]byte{contractIDBlockHeightKeyPrefix}, uint64ToBytes(contractID), uint64ToBytes(blockHeight), uint16ToBytes(eventIndex))
}

func prefixContractIDBlockHight(contractID uint64, blockHeight uint64) []byte {
	return util.PrefixKey([]byte{contractIDBlockHeightKeyPrefix}, uint64ToBytes(contractID), uint64ToBytes(blockHeight))
}

func prefixPluginNameTopic(pluginName string, topic string) []byte {
	return util.PrefixKey([]byte{pluginNameTopicKeyPrefix}, []byte(pluginName), []byte(topic))
}

func prefixLastContractID() []byte {
	return util.PrefixKey([]byte{lastContractIDKeyPrefix}, []byte("contract-id"))
}

func uint64ToBytes(n uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	return buf
}

func uint16ToBytes(n uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, n)
	return buf
}

func bytesToUint64(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}
