package events

import (
	"encoding/json"
	"testing"

	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
)

func TestDBIndexerSendEvents(t *testing.T) {
	var tests = []struct {
		blockHeight uint64
		eventData   types.EventData
		eror        error
	}{
		{
			blockHeight: 1,
			eventData:   types.EventData{},
		},
		{
			blockHeight: 2,
			eventData:   types.EventData{},
		},
	}

	mockEventStore := store.NewMockEventStore(store.NewMemStore())
	var dispatcher = &DBIndexerEventDispatcher{EventStore: mockEventStore}

	for _, test := range tests {
		msg, err := json.Marshal(test.eventData)
		require.Nil(t, err)
		err = dispatcher.Send(test.blockHeight, msg)
		require.Equal(t, err, test.eror)
	}
}

func TestDBIndexerGenUniquContractID(t *testing.T) {
	var tests = []struct {
		blockHeight uint64
		eventData   types.EventData
		wantID      uint64
	}{
		{
			blockHeight: 1,
			eventData: types.EventData{
				PluginName: "plugin1",
			},
			wantID: 1,
		},
		{
			blockHeight: 2,
			eventData: types.EventData{
				PluginName: "plugin2",
			},
			wantID: 2,
		},
		{
			blockHeight: 2,
			eventData: types.EventData{
				PluginName: "plugin2",
			},
			wantID: 2,
		},
		{
			blockHeight: 2,
			eventData: types.EventData{
				PluginName: "plugin10",
			},
			wantID: 3,
		},
		{
			blockHeight: 2,
			eventData: types.EventData{
				PluginName: "plugin5",
			},
			wantID: 4,
		},
	}

	mockEventStore := store.NewMockEventStore(store.NewMemStore())
	var dispatcher = &DBIndexerEventDispatcher{EventStore: mockEventStore}

	for _, test := range tests {
		msg, err := json.Marshal(test.eventData)
		require.Nil(t, err)
		err = dispatcher.Send(test.blockHeight, msg)
		require.Nil(t, err)
	}

	for _, test := range tests {
		id, err := mockEventStore.GetContractID(test.eventData.PluginName)
		require.Nil(t, err)
		require.Equal(t, test.wantID, id)
	}
}
