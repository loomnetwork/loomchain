package events

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
)

var (
	valPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	valPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
)

func TestDBIndexerSendEvents(t *testing.T) {
	var tests = []struct {
		blockHeight uint64
		eventData   types.EventData
		eror        error
	}{
		{
			blockHeight: 1,
			eventData: types.EventData{
				PluginName: "plugin1",
			},
		},
		{
			blockHeight: 2,
			eventData: types.EventData{
				PluginName: "plugin1",
			},
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

func TestDBIndexerGenUniqueContractID(t *testing.T) {
	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	pubKey2, _ := hex.DecodeString(valPubKeyHex2)

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
			blockHeight: 3,
			eventData: types.EventData{
				PluginName: "plugin2",
			},
			wantID: 2,
		},
		{
			blockHeight: 4,
			eventData: types.EventData{
				PluginName: "plugin10",
			},
			wantID: 3,
		},
		{
			blockHeight: 5,
			eventData: types.EventData{
				PluginName: "plugin5",
			},
			wantID: 4,
		},
		{
			blockHeight: 6,
			eventData: types.EventData{
				PluginName: "",
				Address: loom.Address{
					ChainID: "default",
					Local:   loom.LocalAddressFromPublicKey(pubKey1),
				}.MarshalPB(),
			},
			wantID: 5,
		},
		{
			blockHeight: 7,
			eventData: types.EventData{
				PluginName: "",
				Address: loom.Address{
					ChainID: "default",
					Local:   loom.LocalAddressFromPublicKey(pubKey2),
				}.MarshalPB(),
			},
			wantID: 6,
		},
		{
			blockHeight: 8,
			eventData: types.EventData{
				PluginName: "",
				Address: loom.Address{
					ChainID: "default",
					Local:   loom.LocalAddressFromPublicKey(pubKey1),
				}.MarshalPB(),
			},
			wantID: 5,
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
		var name = test.eventData.PluginName
		if test.eventData.PluginName == "" {
			name = loom.UnmarshalAddressPB(test.eventData.Address).String()
		}
		id, err := mockEventStore.GetContractID(name)
		require.Nil(t, err)
		require.Equal(t, test.wantID, id)
	}
}
