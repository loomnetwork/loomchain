package pubsub

import "sync"

// hub implements Hub interface
// Remembers which subscribers messages have been published to
// and does not send repeat messages to any subscribers.
// Revert resets the memory of the subscribers that have received messages.
type resetHub struct {
	mutex    *sync.RWMutex
	registry map[Subscriber]bool
}

// New returns new hub instance. hub is goroutine safe.
func NewResetHub() (result ResetHub) {
	result = &resetHub{
		mutex:    &sync.RWMutex{},
		registry: map[Subscriber]bool{},
	}

	return
}

// CloseSubscriber removes subscriber from hub
func (h *resetHub) CloseSubscriber(subscriber Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.registry, subscriber)
}

// Publish publishes message to subscribers
// todo Warning this function can throw an exception
func (h *resetHub) Publish(message Message) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	count := 0
	wg := &sync.WaitGroup{}

	// iterate over all subscribers, and publish message in separate goroutines
	for sub, unsent := range h.registry {
		if unsent {
			if sub.Match(message.Topic()) {
				wg.Add(1)
				go func() {
					count += sub.Publish(message)
					wg.Done()
				}()
			}
			h.registry[sub] = false
		}
	}

	wg.Wait()

	return count
}

// Subscribe adds subscription to topics and returns subscriber
func (h *resetHub) Subscribe(topics ...string) Subscriber {
	result := newSubscriber(h).Subscribe(topics...)

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.registry[result] = true

	return result
}

func (h *resetHub) Reset() {
	for sub := range h.registry {
		h.registry[sub] = true
	}
}
