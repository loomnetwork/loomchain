package pubsub

import "sync"


var countMu sync.Mutex

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
// todo Warning this function can throw an exception
func (h *hub) Publish(message Message) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
       
	//var count uint64 = 0

        count := 0
	wg := &sync.WaitGroup{}

	// iterate over all subscribers, and publish message in separate goroutines
	for subscriber := range h.registry {
		sub := subscriber
		if sub.Match(message.Topic()) {
			wg.Add(1)
			go func(mRoutine Message) {
				countMu.Lock() 
                                count += sub.Publish(mRoutine)
				countMu.Unlock()
                               // atomic.AddUint64(&count, sub.Publish(mRoutine))
                                wg.Done()
			}(message)
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
