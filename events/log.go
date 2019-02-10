package events

import "log"

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
