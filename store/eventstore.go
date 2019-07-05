package store

import (
	"encoding/binary"
	"sync"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	blockHeightKeyPrefix           byte = 1
	pluginNameKeyPrefix            byte = 2
	contractIDBlockHeightKeyPrefix byte = 3
	lastContractIDKeyPrefix             = 5
)

type EventFilter struct {
	FromBlock uint64
	ToBlock   uint64
	Contract  string
}

type EventStore interface {
	// SaveEvent save a single event
	// contractID, blockHeight, and eventIndex are required for making a unique keys
	SaveEvent(contractID uint64, blockHeight uint64, eventIndex uint16, eventData *types.EventData) error
	// BatchSaveEvents save series of events
	BatchSaveEvents(events []*types.EventData) error
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

func (s *KVEventStore) BatchSaveEvents(events []*types.EventData) error {
	// assume eevents is already sorted by event index
	batch := s.NewBatch()
	for i, event := range events {
		// resolve contractID
		// Go contract uses plugin name, EVM contract uses address
		var contractID uint64
		contractName := event.PluginName
		if contractName == "" {
			contractName = loom.UnmarshalAddressPB(event.Address).String()
		}
		contractID = s.GetContractID(contractName)

		data, err := proto.Marshal(event)
		if err != nil {
			return err
		}
		eventIndex := uint16(i)
		batch.Set(prefixBlockHeightEventIndex(event.BlockHeight, eventIndex), data)
		batch.Set(prefixContractIDBlockHightEventIndex(contractID, event.BlockHeight, eventIndex), data)
	}
	batch.Write()
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

func prefixBlockHeightEventIndex(blockHeight uint64, eventIndex uint16) []byte {
	return util.PrefixKey([]byte{blockHeightKeyPrefix}, uint64ToBytes(blockHeight), uint16ToBytes(eventIndex))
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
