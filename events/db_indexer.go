package events

import (
	"encoding/json"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

type DBIndexerEventDispatcher struct {
	store.EventStore
}

var _ loomchain.EventDispatcher = &DBIndexerEventDispatcher{}

func NewDBIndexerEventDispatcher(es store.EventStore) *DBIndexerEventDispatcher {
	return &DBIndexerEventDispatcher{EventStore: es}
}

func (ed *DBIndexerEventDispatcher) Send(blockHeight uint64, eventIndex int, msg []byte) error {
	var eventData types.EventData
	var err error
	if err = json.Unmarshal(msg, &eventData); err != nil {
		return err
	}

	// resolve contractID
	// Go contract uses plugin name, EVM contract uses address
	var contractID uint64
	contractName := eventData.PluginName
	if contractName == "" {
		contractName = loom.UnmarshalAddressPB(eventData.Address).String()
	}
	contractID = ed.GetContractID(contractName)

	// event index should fit in uint16
	if err := ed.SaveEvent(contractID, eventData.BlockHeight, uint16(eventIndex), &eventData); err != nil {
		return err
	}

	return nil
}

func Query(es store.EventStore, filter store.EventFilter) ([]*types.EventData, error) {
	return es.FilterEvents(filter)
}
