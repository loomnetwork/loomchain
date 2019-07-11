package store

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestEventStoreSetMemDB(t *testing.T) {
	memdb := dbm.NewMemDB()
	var eventStore EventStore = NewKVEventStore(memdb)

	// set pluginname
	contractID := eventStore.GetContractID("plugin1")
	require.EqualValues(t, 1, contractID)
	val := memdb.Get(prefixPluginName("plugin1"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	contractID = eventStore.GetContractID("plugin2")
	val = memdb.Get(prefixPluginName("plugin2"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	event1 := types.EventData{BlockHeight: 1, EncodedBody: []byte("event1")}
	err := eventStore.SaveEvent(contractID, event1.BlockHeight, uint16(event1.TransactionIndex), &event1)
	require.Nil(t, err)
	val = memdb.Get(prefixBlockHeightEventIndex(event1.BlockHeight, uint16(event1.TransactionIndex)))
	var gotevent1 types.EventData
	err = proto.Unmarshal(val, &gotevent1)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event1, &gotevent1))

	event2 := types.EventData{BlockHeight: 2, EncodedBody: []byte("event2")}
	err = eventStore.SaveEvent(contractID, event2.BlockHeight, uint16(event2.TransactionIndex), &event2)
	require.Nil(t, err)
	val = memdb.Get(prefixBlockHeightEventIndex(event2.BlockHeight, uint16(event2.TransactionIndex)))
	var gotevent2 types.EventData
	err = proto.Unmarshal(val, &gotevent2)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event2, &gotevent2))

	event3 := types.EventData{BlockHeight: 20, TransactionIndex: 0, EncodedBody: []byte("event3")}
	err = eventStore.SaveEvent(contractID, event3.BlockHeight, uint16(event3.TransactionIndex), &event3)
	require.Nil(t, err)
	val = memdb.Get(prefixContractIDBlockHightEventIndex(contractID, event3.BlockHeight, uint16(event3.TransactionIndex)))
	var gotevent3 types.EventData
	err = proto.Unmarshal(val, &gotevent3)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event3, &gotevent3))

	event4 := types.EventData{BlockHeight: 20, TransactionIndex: 1, EncodedBody: []byte("event4")}
	err = eventStore.SaveEvent(contractID, event4.BlockHeight, uint16(event4.TransactionIndex), &event4)
	require.Nil(t, err)
	val = memdb.Get(prefixContractIDBlockHightEventIndex(contractID, event4.BlockHeight, uint16(event4.TransactionIndex)))
	var gotevent4 types.EventData
	err = proto.Unmarshal(val, &gotevent4)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event4, &gotevent4))
}

func TestEventStoreFilterMemDB(t *testing.T) {
	memdb := dbm.NewMemDB()
	var eventStore EventStore = NewKVEventStore(memdb)
	contractID := eventStore.GetContractID("plugin1")
	require.EqualValues(t, 1, contractID)

	var blockHeight1 uint64 = 1
	var blockHeight2 uint64 = 2

	var eventData1 []*types.EventData
	for i := 0; i < 10; i++ {
		event := types.EventData{
			BlockHeight:      blockHeight1,
			TransactionIndex: uint64(i),
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", blockHeight1, i)),
		}
		eventStore.SaveEvent(contractID, blockHeight1, uint16(event.TransactionIndex), &event)
		eventData1 = append(eventData1, &event)
	}
	// more event for testing filter
	var eventData2 []*types.EventData
	for i := 0; i < 15; i++ {
		event := types.EventData{
			BlockHeight:      blockHeight2,
			TransactionIndex: uint64(i),
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", blockHeight2, i)),
		}
		eventStore.SaveEvent(contractID, blockHeight2, uint16(event.TransactionIndex), &event)
		eventData2 = append(eventData2, &event)
	}

	filter1 := EventFilter{
		FromBlock: 1,
		ToBlock:   1,
		Contract:  "plugin1",
	}
	events, err := eventStore.FilterEvents(filter1)
	require.Nil(t, err)
	require.Equal(t, len(eventData1), len(events), "expect the same length")
	for i, e := range events {
		require.True(t, proto.Equal(eventData1[i], e))
	}

	filter2 := EventFilter{
		FromBlock: 2,
		ToBlock:   2,
		Contract:  "plugin1",
	}
	events, err = eventStore.FilterEvents(filter2)
	require.Nil(t, err)
	require.Equal(t, len(eventData2), len(events), "expect the same length")
	for i, e := range events {
		require.True(t, proto.Equal(eventData2[i], e))
	}

	filter3 := EventFilter{
		FromBlock: 1,
		ToBlock:   2,
		Contract:  "plugin1",
	}
	events, err = eventStore.FilterEvents(filter3)
	require.Nil(t, err)
	require.Equal(t, len(eventData1)+len(eventData2), len(events), "expect the same length")
	var allEventData []*types.EventData
	allEventData = append(allEventData, eventData1...)
	allEventData = append(allEventData, eventData2...)
	for i, e := range events {
		require.True(t, proto.Equal(allEventData[i], e))
	}
}

func TestEventStoreFilterMultiplePluginsMemDB(t *testing.T) {
	memdb := dbm.NewMemDB()
	var eventStore EventStore = NewKVEventStore(memdb)

	contractID1 := eventStore.GetContractID("plugin1")
	require.EqualValues(t, 1, contractID1)
	contractID2 := eventStore.GetContractID("plugin2")
	require.EqualValues(t, 2, contractID2)

	var eventData1 []*types.EventData
	var eventData2 []*types.EventData
	for i := 1; i <= 5; i++ {
		event := types.EventData{
			PluginName:       "plugin1",
			BlockHeight:      uint64(i),
			TransactionIndex: 0,
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", uint64(i), 0)),
		}
		eventStore.SaveEvent(contractID1, uint64(i), uint16(event.TransactionIndex), &event)
		eventData1 = append(eventData1, &event)

		event2 := types.EventData{
			PluginName:       "plugin2",
			BlockHeight:      uint64(i),
			TransactionIndex: 1,
			EncodedBody:      []byte(fmt.Sprintf("event2-%d-%d", uint64(i), 0)),
		}
		eventStore.SaveEvent(contractID2, uint64(i), uint16(event2.TransactionIndex), &event2)
		eventData2 = append(eventData2, &event2)
	}

	for i := 6; i <= 10; i++ {
		event := types.EventData{
			PluginName:       "plugin2",
			BlockHeight:      uint64(i),
			TransactionIndex: 0,
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", uint64(i), 0)),
		}
		eventStore.SaveEvent(contractID2, uint64(i), uint16(event.TransactionIndex), &event)
		eventData2 = append(eventData2, &event)
	}

	filter1 := EventFilter{
		FromBlock: 1,
		ToBlock:   10,
		Contract:  "plugin1",
	}
	events, err := eventStore.FilterEvents(filter1)
	require.Nil(t, err)
	require.Equal(t, len(eventData1), len(events), "expect the same length")
	for i, e := range events {
		require.True(t, proto.Equal(eventData1[i], e))
	}

	filter2 := EventFilter{
		FromBlock: 1,
		ToBlock:   10,
		Contract:  "plugin2",
	}
	events, err = eventStore.FilterEvents(filter2)
	require.Nil(t, err)
	require.Equal(t, len(eventData2), len(events), "expect the same length")
	for i, e := range events {
		require.True(t, proto.Equal(eventData2[i], e))
	}

	for i := 11; i <= 15; i++ {
		event := types.EventData{
			PluginName:       "plugin1",
			BlockHeight:      uint64(i),
			TransactionIndex: 0,
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", uint64(i), 0)),
		}
		eventStore.SaveEvent(contractID1, uint64(i), uint16(event.TransactionIndex), &event)
		eventData1 = append(eventData1, &event)
	}

	filter3 := EventFilter{
		FromBlock: 1,
		ToBlock:   15,
		Contract:  "plugin1",
	}
	events, err = eventStore.FilterEvents(filter3)
	require.Nil(t, err)
	require.Equal(t, len(eventData1), len(events), "expect the same length")
	for i, e := range events {
		require.True(t, proto.Equal(eventData1[i], e))
	}

	filter4 := EventFilter{
		FromBlock: 1,
		ToBlock:   15,
	}
	_, err = eventStore.FilterEvents(filter4)
	require.Nil(t, err)
}
