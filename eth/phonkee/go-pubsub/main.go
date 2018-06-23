package pubsub

import (
	"encoding/json"
	"sync"
)

var (
	// Provide default hub
	defaultHub Hub
)

func init() {
	defaultHub = New()
}

// New returns new hub instance. hub is goroutine safe.
func New() (result Hub) {
	result = &hub{
		mutex:    &sync.RWMutex{},
		registry: map[Subscriber]int{},
	}

	return
}

// NewMessage returns new message to be published
// body can be any type, if it's []byte it's directly used. If it's string we convert it to []byte
// all other types are json marshaled
func NewMessage(topic string, body interface{}) Message {
	result := &message{
		TopicValue: topic,
	}

	switch body := body.(type) {
	case []byte:
		result.BodyValue = body
	case string:
		result.BodyValue = []byte(body)
	default:
		result.BodyValue, _ = json.Marshal(body)
	}

	return result
}

// Publish publishes message on default Hub instance
func Publish(message Message) int {
	return defaultHub.Publish(message)
}

// Subscribe subscribes to topics on default hub
func Subscribe(topics ...string) Subscriber {
	return defaultHub.Subscribe(topics...)
}
