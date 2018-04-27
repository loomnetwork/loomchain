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
func (ed *LogEventDispatcher) Send(index int64, msg []byte) error {
	log.Printf("Event emitted: index: %d, msg: %s\n", index, msg)
	return nil
}
