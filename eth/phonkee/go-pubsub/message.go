package pubsub

// message
type message struct {
	// BodyValue is body of message. It's exported so we don't need to do custom json parsing
	BodyValue []byte `json:"body"`

	// TopicValue is topic for given message.
	TopicValue string `json:"topic"`
}

// Body returns body of message
func (m *message) Body() []byte {
	return m.BodyValue
}

// Topic returns topic
func (m *message) Topic() string {
	return m.TopicValue
}
