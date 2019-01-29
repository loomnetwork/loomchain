package events

import (
	"encoding/json"
	"log"

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
	var eventData loomchain.EventData
	if err := json.Unmarshal(msg, &eventData); err != nil {
		return err
	}

	// TODO: more efficient way to persist event data
	ed.SetEventByBlockHightEventIndex(eventData.BlockHeight, eventData.TransactionIndex, msg)
	ed.SetEventByPluginName(eventData.PluginName, msg) // EVM use the address of EVM??

	// TODO: Remove printf
	log.Printf("Event emitted: index: %d, eventData: %+v\n", blockHeight, eventData)
	return nil
}
