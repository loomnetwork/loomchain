package eth_pubsub

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/phonkee/go-pubsub"
	"sync"
)

// newSubscriber returns ethSubscriber for given topics
func newEthSubscriber(hub pubsub.ResetHub, topics ...string) (result pubsub.Subscriber) {
	filter, _ := utils.UnmarshalEthFilter([]byte(topics[0]))
	result = &ethSubscriber{
		hub:    hub,
		mutex:  &sync.RWMutex{},
		sf:     nil,
		filter: filter.EthBlockFilter,
	}
	return result.Subscribe(topics...)
}

// ethSubscriber is Subscriber implementation
type ethSubscriber struct {
	hub    pubsub.ResetHub
	mutex  *sync.RWMutex
	sf     pubsub.SubscriberFunc
	filter utils.EthBlockFilter
}

// Close ethSubscriber removes ethSubscriber from hub and stops receiving messages
func (s *ethSubscriber) Close() {
	s.hub.CloseSubscriber(s)
}

// Do sets ethSubscriber function that will be called when message arrives
func (s *ethSubscriber) Do(sf pubsub.SubscriberFunc) pubsub.Subscriber {
	s.sf = sf
	return s
}

// Match returns whether ethSubscriber topics matches
func (s *ethSubscriber) Match(topic string) bool {
	events := types.EventData{}
	if err := proto.Unmarshal([]byte(topic), &events); err != nil {
		return false
	}

	return false // utils.MatchEthFilter(s.filter, events)
}

// Publish publishes message to ethSubscriber
func (s *ethSubscriber) Publish(message pubsub.Message) int {
	if s.sf == nil {
		return 0
	}
	s.sf(message)
	return 1
}

// Subscribe subscribes to topics
func (s *ethSubscriber) Subscribe(topics ...string) pubsub.Subscriber {
	panic("should never be called")
	return s
}

// Topics returns whole list of all topics subscribed to
func (s *ethSubscriber) Topics() []string {
	panic("should never be called")
	return []string{}
}

// Unsubscribe unsubscribes from given topics (exact match)
func (s *ethSubscriber) Unsubscribe(topics ...string) pubsub.Subscriber {
	panic("should never be called")
	return s
}
