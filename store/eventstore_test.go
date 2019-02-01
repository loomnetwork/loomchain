package store

import (
	"fmt"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	cdb "github.com/loomnetwork/loomchain/db"
	"github.com/stretchr/testify/require"
)

func TestEventStoreSetMemDB(t *testing.T) {
	memstore := NewMemStore()
	var eventStore EventStore = NewMockEventStore(memstore)

	// set pluginname
	var contractID uint64 = 1
	contractID = eventStore.GetContractID("plugin1")
	require.EqualValues(t, 1, contractID)
	val := memstore.Get(prefixPluginName("plugin1"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	contractID = eventStore.GetContractID("plugin2")
	val = memstore.Get(prefixPluginName("plugin2"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	event1 := types.EventData{BlockHeight: 1, EncodedBody: []byte("event1")}
	err := eventStore.SaveEvent(contractID, event1.BlockHeight, uint16(event1.TransactionIndex), &event1)
	require.Nil(t, err)
	val = memstore.Get(prefixBlockHeightEventIndex(event1.BlockHeight, uint16(event1.TransactionIndex)))
	var gotevent1 types.EventData
	err = proto.Unmarshal(val, &gotevent1)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event1, &gotevent1))

	event2 := types.EventData{BlockHeight: 2, EncodedBody: []byte("event2")}
	err = eventStore.SaveEvent(contractID, event2.BlockHeight, uint16(event2.TransactionIndex), &event2)
	require.Nil(t, err)
	val = memstore.Get(prefixBlockHeightEventIndex(event2.BlockHeight, uint16(event2.TransactionIndex)))
	var gotevent2 types.EventData
	err = proto.Unmarshal(val, &gotevent2)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event2, &gotevent2))

	event3 := types.EventData{BlockHeight: 20, TransactionIndex: 0, EncodedBody: []byte("event3")}
	err = eventStore.SaveEvent(contractID, event3.BlockHeight, uint16(event3.TransactionIndex), &event3)
	require.Nil(t, err)
	val = memstore.Get(prefixContractIDBlockHightEventIndex(contractID, event3.BlockHeight, uint16(event3.TransactionIndex)))
	var gotevent3 types.EventData
	err = proto.Unmarshal(val, &gotevent3)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event3, &gotevent3))

	event4 := types.EventData{BlockHeight: 20, TransactionIndex: 1, EncodedBody: []byte("event4")}
	err = eventStore.SaveEvent(contractID, event4.BlockHeight, uint16(event4.TransactionIndex), &event4)
	require.Nil(t, err)
	val = memstore.Get(prefixContractIDBlockHightEventIndex(contractID, event4.BlockHeight, uint16(event4.TransactionIndex)))
	var gotevent4 types.EventData
	err = proto.Unmarshal(val, &gotevent4)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event4, &gotevent4))
}
func TestEventStoreSetLevelDB(t *testing.T) {
	dbpath := os.TempDir()
	db, err := cdb.LoadDB("goleveldb", "event", dbpath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbpath)

	var eventStore EventStore = NewKVEventStore(db)

	// set pluginname
	var contractID uint64 = 1
	contractID = eventStore.GetContractID("plugin1")
	require.EqualValues(t, 1, contractID)
	val := db.Get(prefixPluginName("plugin1"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	contractID = eventStore.GetContractID("plugin2")
	val = db.Get(prefixPluginName("plugin2"))
	require.EqualValues(t, contractID, bytesToUint64(val))

	event1 := types.EventData{BlockHeight: 1, EncodedBody: []byte("event1")}
	err = eventStore.SaveEvent(contractID, event1.BlockHeight, uint16(event1.TransactionIndex), &event1)
	require.Nil(t, err)
	val = db.Get(prefixBlockHeightEventIndex(event1.BlockHeight, uint16(event1.TransactionIndex)))
	var gotevent1 types.EventData
	err = proto.Unmarshal(val, &gotevent1)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event1, &gotevent1))

	event2 := types.EventData{BlockHeight: 2, EncodedBody: []byte("event2")}
	err = eventStore.SaveEvent(contractID, event2.BlockHeight, uint16(event2.TransactionIndex), &event2)
	require.Nil(t, err)
	val = db.Get(prefixBlockHeightEventIndex(event2.BlockHeight, uint16(event2.TransactionIndex)))
	var gotevent2 types.EventData
	err = proto.Unmarshal(val, &gotevent2)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event2, &gotevent2))

	event3 := types.EventData{BlockHeight: 20, TransactionIndex: 0, EncodedBody: []byte("event3")}
	err = eventStore.SaveEvent(contractID, event3.BlockHeight, uint16(event3.TransactionIndex), &event3)
	require.Nil(t, err)
	val = db.Get(prefixContractIDBlockHightEventIndex(contractID, event3.BlockHeight, uint16(event3.TransactionIndex)))
	var gotevent3 types.EventData
	err = proto.Unmarshal(val, &gotevent3)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event3, &gotevent3))

	event4 := types.EventData{BlockHeight: 20, TransactionIndex: 1, EncodedBody: []byte("event4")}
	err = eventStore.SaveEvent(contractID, event4.BlockHeight, uint16(event4.TransactionIndex), &event4)
	require.Nil(t, err)
	val = db.Get(prefixContractIDBlockHightEventIndex(contractID, event4.BlockHeight, uint16(event4.TransactionIndex)))
	var gotevent4 types.EventData
	err = proto.Unmarshal(val, &gotevent4)
	require.Nil(t, err)
	require.True(t, proto.Equal(&event4, &gotevent4))
}

func TestEventStoreFilterMemDB(t *testing.T) {
	memstore := NewMemStore()
	var eventStore EventStore = NewMockEventStore(memstore)
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

func TestEventStoreFilterLevelDB(t *testing.T) {
	dbpath := os.TempDir()
	db, err := cdb.LoadDB("goleveldb", "event", dbpath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbpath)

	var eventStore EventStore = NewKVEventStore(db)
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

func TestEventStoreFilterMultiplePluginsLevelDB(t *testing.T) {
	dbpath := os.TempDir()
	db, err := cdb.LoadDB("goleveldb", "event", dbpath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbpath)

	var eventStore EventStore = NewKVEventStore(db)
	contractID1 := eventStore.GetContractID("plugin1")
	require.EqualValues(t, uint64(1), contractID1)
	contractID2 := eventStore.GetContractID("plugin2")
	require.EqualValues(t, uint64(2), contractID2)

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
	events, err = eventStore.FilterEvents(filter4)
	require.Nil(t, err)
	require.Equal(t, len(eventData1)+len(eventData2), len(events), "expect the same length")
}

func TestEventStoreFilterMultiplePluginsMemDB(t *testing.T) {
	memstore := NewMemStore()
	var eventStore EventStore = NewMockEventStore(memstore)

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
	events, err = eventStore.FilterEvents(filter4)
	require.Nil(t, err)
}

func BenchmarkEventStoreFilterLevelDB(b *testing.B) {
	dbpath := os.TempDir()
	db, err := cdb.LoadDB("goleveldb", "event", dbpath)
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dbpath)

	var eventStore EventStore = NewKVEventStore(db)
	contractID := eventStore.GetContractID("plugin1")

	// populate 100 blocks, 10 events in each
	for h := uint64(1); h <= 100; h++ {
		for i := uint64(0); i < 10; i++ {
			event := types.EventData{
				BlockHeight:      h,
				TransactionIndex: i,
				EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", h, i)),
			}
			eventStore.SaveEvent(contractID, h, uint16(event.TransactionIndex), &event)
		}
	}

	// benchmarks to test
	benchmarks := []struct {
		fromBlock uint64
		toBlock   uint64
	}{
		{1, 10}, {1, 20}, {1, 30}, {1, 50}, {1, 70}, {1, 90},
	}

	for _, bm := range benchmarks {
		bmName := fmt.Sprintf("BM FilterEvents from %d to %d", bm.fromBlock, bm.toBlock)
		b.Run(bmName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				filter := EventFilter{
					FromBlock: bm.fromBlock,
					ToBlock:   bm.toBlock,
					Contract:  "plugin1",
				}
				_, err = eventStore.FilterEvents(filter)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

func BenchmarkEventStoreSetEventLevelDB(b *testing.B) {
	benchmarks := []struct {
		numBlocks      uint64
		eventsPerBlock uint64
	}{
		{1, 10},
		{1, 25},
		{1, 50},
		{1, 75},
		{1, 100},
		{1, 250},
		{1, 500},
	}

	for _, bm := range benchmarks {
		bmName := fmt.Sprintf("BM SetEvent numBlocks %d eventsPerBlock %d", bm.numBlocks, bm.eventsPerBlock)
		b.Run(bmName, func(b *testing.B) {
			dbpath := os.TempDir()
			db, err := cdb.LoadDB("goleveldb", "event", dbpath)
			if err != nil {
				b.Fatal(err)
			}
			defer os.RemoveAll(dbpath)

			var eventStore EventStore = NewKVEventStore(db)
			contractID := eventStore.GetContractID("plugin1")
			for n := 0; n < b.N; n++ {

				// populate `numBlocks` blocks, `eventsPerBlock` events in each
				for h := uint64(1); h <= bm.numBlocks; h++ {
					for i := uint64(0); i < bm.eventsPerBlock; i++ {
						event := types.EventData{
							BlockHeight:      h,
							TransactionIndex: i,
							EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", h, i)),
						}
						err = eventStore.SaveEvent(contractID, h, uint16(event.TransactionIndex), &event)
						if err != nil {
							b.Error(err)
						}
					}
				}
			}
		})
	}

}
