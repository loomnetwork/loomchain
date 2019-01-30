package events

import (
	"encoding/json"
	"log"

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

func (ed *DBIndexerEventDispatcher) Send(blockHeight uint64, msg []byte) error {
	var eventData types.EventData
	var err error
	if err = json.Unmarshal(msg, &eventData); err != nil {
		return err
	}

	// resolve contractID
	// Go contract uses plugin name, EVM contract uses address
	var contractID uint64
	if eventData.PluginName != "" {
		contractID, err = ed.GetContractID(eventData.PluginName)
		if err != nil {
			return err
		}
		if contractID == 0 {
			contractID = ed.NextContractID()
			ed.SetContractID(eventData.PluginName, contractID)
		}
	} else {
		address := loom.UnmarshalAddressPB(eventData.Address).String()
		contractID, err = ed.GetContractID(address)
		if err != nil {
			return err
		}
		if contractID == 0 {
			contractID = ed.NextContractID()
			ed.SetContractID(address, contractID)
		}
		contractID, err = ed.GetContractID(address)
	}

	if err := ed.SetEvent(contractID, eventData.BlockHeight, &eventData); err != nil {
		return err
	}

	// TODO: Remove printf
	log.Printf("Event emitted: index: %d, contractID: %d, eventData: %+v\n", blockHeight, contractID, eventData)
	return nil
}

func Query(es store.EventStore, filter *types.EventFilter) ([]*types.EventData, error) {
	// TODO: validate filter
	return es.FilterEvents(filter)
}
