package events

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

type DBIndexerEventDispatcher struct {
	store.EventStore
	events []*types.EventData
	sync.Mutex
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

	// append the events
	ed.events = append(ed.events, &eventData)

	log.Printf("Event appended: index: %d, msg: %v\n", eventIndex, eventData)
	return nil
}

func (ed *DBIndexerEventDispatcher) Flush() {
	ed.Lock()
	log.Printf("Event batch writing length: %d\n", len(ed.events))
	if err := ed.EventStore.BatchSaveEvents(ed.events); err != nil {
		log.Printf("Event Dispatcher flush error: %s", err)
	}
	ed.events = make([]*types.EventData, 0)
	ed.Unlock()
}

func Query(es store.EventStore, filter store.EventFilter) ([]*types.EventData, error) {
	return es.FilterEvents(filter)
}
