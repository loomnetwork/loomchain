package events

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"log"
)

// LogEventDispatcher just logs events
type LogEventDispatcher struct {
}

// NewLogEventDispatcher create a new redis dispatcher
func NewLogEventDispatcher() *LogEventDispatcher {
	return &LogEventDispatcher{}
}

// Send sends the event
func (ed *LogEventDispatcher) Send(index uint64, msg []byte) error {
	log.Printf("Event emitted: index: %d, length: %d, msg: %s\n", index, len(msg), msg)
	return nil
}

func (ed *LogEventDispatcher) SaveToChain(msgs []*types.EventData, txRes *loomchain.TxHandlerResult) {

}
