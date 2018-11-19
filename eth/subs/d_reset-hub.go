package subs

import (
	"sync"

	"github.com/phonkee/go-pubsub"
)

// hub implements Hub interface
// Remembers which subscribers messages have been published to
// and does not send repeat messages to any subscribers.
// Revert resets the memory of the subscribers that have received messages.
type EthDepreciatedResetHub struct {
	mutex    *sync.RWMutex
	registry map[pubsub.Subscriber]bool
}

// New returns new hub instance. hub is goroutine safe.
func NewEthDepreciatedResetHub() (result pubsub.ResetHub) {
	result = &EthDepreciatedResetHub{
		mutex:    &sync.RWMutex{},
		registry: map[pubsub.Subscriber]bool{},
	}
	return
}

// CloseSubscriber removes subscriber from hub
func (h *EthDepreciatedResetHub) CloseSubscriber(subscriber pubsub.Subscriber) {
	h.mutex.Lock()
	delete(h.registry, subscriber)
	h.mutex.Unlock()
}

// Publish publishes message to subscribers
// todo Warning this function can throw an exception
func (h *EthDepreciatedResetHub) Publish(message pubsub.Message) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	count := 0
	// iterate over all subscribers, and publish messages
	for sub, unsent := range h.registry {
		if unsent {
			if sub.Match(message.Topic()) {
				count += sub.Publish(message)
				h.registry[sub] = false
			}
		}
	}

	return count
}

// Subscribe adds subscription to topics and returns subscriber
func (h *EthDepreciatedResetHub) Subscribe(topics ...string) pubsub.Subscriber {
	var result pubsub.Subscriber
	if len(topics) > 0 {
		result = newEthSubscriber(h, topics[0])
	} else {
		result = newEthSubscriber(h, "")
	}

	h.mutex.Lock()
	h.registry[result] = true
	h.mutex.Unlock()

	return result
}

func (h *EthDepreciatedResetHub) Reset() {
	h.mutex.Lock()
	for sub := range h.registry {
		h.registry[sub] = true
	}
	h.mutex.Unlock()
}