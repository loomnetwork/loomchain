package events

import (
	"encoding/json"
	"testing"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
)

func TestDBIndexerSendEvents(t *testing.T) {
	var tests = []struct {
		blockHeight uint64
		eventData   loomchain.EventData
		eror        error
	}{
		{
			blockHeight: 1,
			eventData:   loomchain.EventData{},
		},
		{
			blockHeight: 2,
			eventData:   loomchain.EventData{},
		},
	}

	var dispatcher = &DBIndexerEventDispatcher{EventStore: store.NewMockEventStore()}

	for _, test := range tests {
		msg, err := json.Marshal(test.eventData)
		require.Nil(t, err)
		err = dispatcher.Send(test.blockHeight, msg)
		require.Equal(t, err, test.eror)
	}
}
