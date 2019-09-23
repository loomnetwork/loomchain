package events

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/loomnetwork/go-loom/plugin/types"

	"github.com/loomnetwork/loomchain/store"
)

type DBIndexerEventDispatcher struct {
	store.EventStore
	events []*types.EventData
	sync.Mutex
}

//var _ loomchain.EventDispatcher = &DBIndexerEventDispatcher{}

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
	ed.Lock()
	ed.events = append(ed.events, &eventData)
	ed.Unlock()

	return nil
}

func (ed *DBIndexerEventDispatcher) Flush() {
	var flushEvents []*types.EventData
	ed.Lock()
	flushEvents = ed.events
	ed.events = make([]*types.EventData, 0)
	ed.Unlock()
	if err := ed.EventStore.BatchSaveEvents(flushEvents); err != nil {
		log.Printf("Event dispatcher flush error: %s", err)
	}
}
