package pubsub

import "sync"

// hub implements Hub interface
type hub struct {
	mutex    *sync.RWMutex
	registry map[Subscriber]int
}

// CloseSubscriber removes subscriber from hub
func (h *hub) CloseSubscriber(subscriber Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.registry, subscriber)
}

// Publish publishes message to subscribers
func (h *hub) Publish(message Message) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	count := 0

	wg := &sync.WaitGroup{}

	// iterate over all subscribers, and publish message in separate goroutines
	for subscriber := range h.registry {
		sub := subscriber
		if sub.Match(message.Topic()) {
			wg.Add(1)
			go func() {
				count += sub.Publish(message)
				wg.Done()
			}()
		}
	}

	wg.Wait()

	return count
}

// Subscribe adds subscription to topics and returns subscriber
func (h *hub) Subscribe(topics ...string) Subscriber {

	result := newSubscriber(h).Subscribe(topics...)

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.registry[result] = 1

	return result
}
