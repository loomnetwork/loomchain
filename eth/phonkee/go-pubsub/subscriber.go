package pubsub

import (
	"strings"
	"sync"
)

// newSubscriber returns subscriber for given topics
func newSubscriber(hub Hub, topics ...string) (result Subscriber) {
	result = &subscriber{
		hub:    hub,
		mutex:  &sync.RWMutex{},
		sf:     nil,
		topics: []string{},
	}

	return result.Subscribe(topics...)
}

// subscriber is Subscriber implementation
type subscriber struct {
	hub    Hub
	mutex  *sync.RWMutex
	sf     SubscriberFunc
	topics []string
}

// Close subscriber removes subscriber from hub and stops receiving messages
func (s *subscriber) Close() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	s.hub.CloseSubscriber(s)
}

// Do sets subscriber function that will be called when message arrives
func (s *subscriber) Do(sf SubscriberFunc) Subscriber {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sf = sf

	return s
}

// Match returns whether subscriber topics matches
func (s *subscriber) Match(topic string) bool {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// iterate over all topics
	for _, st := range s.topics {
		if strings.HasPrefix(topic, st) {

			// same topic
			if len(st) == len(topic) {
				return true
			}

			trimmed := strings.TrimPrefix(topic, st)
			if len(trimmed) > 0 {
				if string(trimmed[0]) == ":" {
					return true
				}
			}

		}
	}

	return false
}

// Publish publishes message to subscriber
func (s *subscriber) Publish(message Message) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.sf == nil {
		return 0
	}

	// call
	s.sf(message)

	return 1
}

// Subscribe subscribes to topics
func (s *subscriber) Subscribe(topics ...string) Subscriber {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newTopics := []string{}

	for _, topic := range topics {
		if stringInSlice(topic, s.topics) {
			if !stringInSlice(topic, newTopics) {
				newTopics = append(newTopics, topic)
			}
		} else {
			newTopics = append(newTopics, topic)
		}
	}

	s.topics = newTopics

	return s
}

// Topics returns whole list of all topics subscribed to
func (s *subscriber) Topics() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.topics
}

// Unsubscribe unsubscribes from given topics (exact match)
func (s *subscriber) Unsubscribe(topics ...string) Subscriber {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	newTopics := make([]string, 0, len(s.topics))

	for _, topic := range s.topics {
		if !stringInSlice(topic, topics) {
			newTopics = append(newTopics, topic)
		}
	}

	s.topics = newTopics

	return s
}
