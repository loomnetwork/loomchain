package events

import (
	"encoding/json"
	"log"

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
	if err := json.Unmarshal(msg, &eventData); err != nil {
		return err
	}

	// TODO: more efficient way to persist event data
	// resolve contractID
	contractID, err := ed.GetContractID(eventData.PluginName)
	if err != nil {
		return err
	}
	// TODO: EVM to find by address
	if contractID == 0 {
		contractID = ed.NextContractID()
		ed.SetContractID(eventData.PluginName, contractID)
	}

	err = ed.SetEventByBlockHightEventIndex(eventData.BlockHeight, eventData.TransactionIndex, msg)
	if err != nil {
		return err
	}
	err = ed.SetEventByContractIDBlockHightEventIndex(contractID, eventData.BlockHeight, eventData.TransactionIndex, msg)
	if err != nil {
		return err
	}

	// TODO: Remove printf
	log.Printf("Event emitted: index: %d, contractID: %d, eventData: %+v\n", blockHeight, contractID, eventData)
	return nil
}
