package pubsub

// SubscriberFunc is function that will be called when message arrives
type SubscriberFunc func(message Message)

// Hub is pubsub main object, on hub you can publish or subscribe to messages.
type Hub interface {

	// CloseSubscriber removes subscriber from hub
	CloseSubscriber(subscriber Subscriber)

	// Publish message
	Publish(message Message) int

	// Status returns stats about hub
	//Stats()

	// Subscribe subscribe to topics
	Subscribe(topics ...string) Subscriber
}

// An extention of the Hub interface that allows for a Reset() method
// that could be used to reinitialise a list of subscribers.
type OnceHub interface {
	Hub

	// Reset subscribers list
	Reset()
}

// Message holds information about topic on which it was sent
type Message interface {

	// Topic returns topic where message was sent
	Topic() string

	// Body returns message body
	Body() []byte
}

type Subscriber interface {

	// Do sets subscriber func
	Do(SubscriberFunc) Subscriber

	// Close closes subscriber
	Close()

	// Publish publishes message
	Publish(message Message) int

	// Match returns whether we should send given message with given topic
	Match(topic string) bool

	// Subscribe subscribe to topics
	Subscribe(topics ...string) Subscriber

	// Unsubscribe removes subscription for given topics
	Unsubscribe(topics ...string) Subscriber

	// Topics returns all topics in which is subscriber interested
	Topics() []string
}
