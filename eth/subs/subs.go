package subs

import (
	"fmt"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/phonkee/go-pubsub"
	"sync"
)

type EthSubscriptionSet struct {
	pubsub.Hub
	clients map[string]pubsub.Subscriber
	callers map[string][]string
	sync.Mutex
}

func NewEthSubscriptionSet() *EthSubscriptionSet {
	s := &EthSubscriptionSet{
		Hub:     pubsub.New(),
		clients: make(map[string]pubsub.Subscriber),
		callers: make(map[string][]string),
	}
	return s
}

func (s *EthSubscriptionSet) For(caller string) (pubsub.Subscriber, string) {
	s.Lock()
	defer s.Unlock()
	id := utils.GetId()
	s.clients[id] = s.Subscribe("system:")
	s.callers[caller] = append(s.callers[caller], id)

	return s.clients[id], id
}

func (s *EthSubscriptionSet) AddSubscription(id, method, filter string) error {
	s.Lock()
	defer s.Unlock()
	if method != "logs" {
		return fmt.Errorf("subscription method \"%s\" not supported", method)
	}

	sub, exists := s.clients[id]
	if !exists {
		return fmt.Errorf("Subscription %s not found", id)
	}

	topics, err := topicsFromFilter(filter)
	if err != nil {
		return err
	}
	sub.Subscribe(append(sub.Topics(), topics...)...)
	return nil
}

func (s *EthSubscriptionSet) Purge(caller string) {
	s.Lock()
	defer s.Unlock()
	if ids, found := s.callers[caller]; found {
		for _, id := range ids {
			if c, ok := s.clients[id]; ok {
				s.CloseSubscriber(c)
				delete(s.clients, id)
			}
		}
		delete(s.callers, caller)
	}
}

func (s *EthSubscriptionSet) Remove(id string) error {
	s.Lock()
	defer s.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return fmt.Errorf("Subscription not found")
	}
	s.CloseSubscriber(c)
	delete(s.clients, id)

	return nil
}

func topicsFromFilter(filter string) ([]string, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topicList := range ethFilter.Topics {
		if len(topicList) > 0 {
			for _, topic := range topicList {
				topics = append(topics, topic)
			}

		}
	}
	for _, addr := range ethFilter.Addresses {
		topics = append(topics, "contract:"+addr.String())
	}
	return topics, nil
}
