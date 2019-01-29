package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventStore(t *testing.T) {
	memstore := NewMemStore()
	var eventStore EventStore = NewMockEventStore(memstore)

	// set pluginname
	err := eventStore.SetContractID("plugin1", 1)
	require.Nil(t, err)
	val := memstore.Get(prefixPluginName("plugin1"))
	require.EqualValues(t, []byte("1"), val)

	err = eventStore.SetContractID("plugin2", 2)
	require.Nil(t, err)
	val = memstore.Get(prefixPluginName("plugin2"))
	require.EqualValues(t, []byte("2"), val)

	err = eventStore.SetContractID("", 999)
	require.Nil(t, err)
	val = memstore.Get(prefixPluginName(""))
	require.EqualValues(t, []byte("999"), val)

	// SetEventByBlockHightEventIndex
	event1 := []byte("event1")
	err = eventStore.SetEventByBlockHightEventIndex(1, 0, event1)
	require.Nil(t, err)
	val = memstore.Get(prefixBlockHightEventIndex(1, 0))
	require.EqualValues(t, event1, val)

	event2 := []byte("somelongerdata")
	err = eventStore.SetEventByBlockHightEventIndex(100, 0, event2)
	require.Nil(t, err)
	val = memstore.Get(prefixBlockHightEventIndex(100, 0))
	require.EqualValues(t, event2, val)

	// SetEventByContractIDBlockHightEventIndex
	event3 := []byte("event3")
	err = eventStore.SetEventByContractIDBlockHightEventIndex(20, 1, 0, event3)
	require.Nil(t, err)
	val = memstore.Get(prefixContractIDBlockHightEventIndex(20, 1, 0))
	require.EqualValues(t, event3, val)

	event4 := []byte("event4")
	err = eventStore.SetEventByContractIDBlockHightEventIndex(4, 100, 0, event4)
	require.Nil(t, err)
	val = memstore.Get(prefixContractIDBlockHightEventIndex(4, 100, 0))
	require.EqualValues(t, event4, val)
}
