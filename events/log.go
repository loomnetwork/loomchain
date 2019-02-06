package events

import (
	"log"

	"github.com/loomnetwork/loomchain"
)

// LogEventDispatcher just logs events
type LogEventDispatcher struct {
}

var _ loomchain.EventDispatcher = &LogEventDispatcher{}

// NewLogEventDispatcher create a new redis dispatcher
func NewLogEventDispatcher() *LogEventDispatcher {
	return &LogEventDispatcher{}
}

// Send sends the event
func (ed *LogEventDispatcher) Send(index uint64, eventIdex int, msg []byte) error {
	log.Printf("Event emitted: index: %d, length: %d, msg: %s\n", index, len(msg), msg)
	return nil
}

func (ed *LogEventDispatcher) Flush() {
}
