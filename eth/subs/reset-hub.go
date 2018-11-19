package subs

import (
	"fmt"
	"sync"

	"github.com/phonkee/go-pubsub"
)

// hub implements Hub interface
// Remembers which subscribers messages have been published to
// and does not send repeat messages to any subscribers.
// Revert resets the memory of the subscribers that have received messages.
type ethResetHub struct {
	mutex    *sync.RWMutex
	registry map[pubsub.Subscriber]bool
}

func newEthResetHub() *ethResetHub {
	return &ethResetHub{
		mutex:    &sync.RWMutex{},
		//registry: map[pubsub.Subscriber]bool{},
		registry: make(map[pubsub.Subscriber]bool),
	}
}

// CloseSubscriber removes subscriber from hub
func (h *ethResetHub) CloseSubscriber(subscriber pubsub.Subscriber) {
	h.mutex.Lock()
	delete(h.registry, subscriber)
	h.mutex.Unlock()
}

// Publish publishes message to subscribers
func (h *ethResetHub) Publish(message pubsub.Message) int {
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

func (h *ethResetHub) addSubscriber(sub pubsub.Subscriber) {
	defer func() {
		if r := recover(); r != nil {
			 fmt.Println("caught panic publishing event: %v", r)
		}
	}()
	h.registry[sub] = true
	fmt.Println("set registry value")
}

// Subscribe adds subscription to topics and returns subscriber
func (h *ethResetHub) Subscribe(_ ...string) pubsub.Subscriber {
	panic("should never be called")
	return nil
}

func (h *ethResetHub) Reset() {
	h.mutex.Lock()
	for sub := range h.registry {
		h.registry[sub] = true
	}
	h.mutex.Unlock()
}

type EthDepreciatedResetHub struct {
	ethResetHub
}

func NewEthDepreciatedResetHub() (result pubsub.ResetHub) {
	result = &EthDepreciatedResetHub{
		ethResetHub: ethResetHub{
			mutex:    &sync.RWMutex{},
			registry: map[pubsub.Subscriber]bool{},
		},
	}
	return
}

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

