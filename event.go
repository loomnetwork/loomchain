package loomchain

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/abci/backend"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/log"
	"github.com/phonkee/go-pubsub"
)

type EventData types.EventData

type EventHandler interface {
	Post(state State, e *EventData) error
	EmitBlockTx(height uint64) error
	SubscriptionSet() *SubscriptionSet
	EthSubscriptionSet() *subs.EthSubscriptionSet
}

type EventDispatcher interface {
	Send(index uint64, msg []byte) error
}

type DefaultEventHandler struct {
	dispatcher       EventDispatcher
	stash            *stash
	backend          backend.Backend
	subscriptions    *SubscriptionSet
	ethSubscriptions *subs.EthSubscriptionSet
}

func NewDefaultEventHandler(dispatcher EventDispatcher, b backend.Backend) *DefaultEventHandler {
	return &DefaultEventHandler{
		dispatcher:       dispatcher,
		stash:            newStash(),
		backend:          b,
		subscriptions:    NewSubscriptionSet(),
		ethSubscriptions: subs.NewEthSubscriptionSet(),
	}
}

func (ed *DefaultEventHandler) SubscriptionSet() *SubscriptionSet {
	return ed.subscriptions
}

func (ed *DefaultEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return ed.ethSubscriptions
}

func (ed *DefaultEventHandler) Post(state State, msg *EventData) error {
	height := uint64(state.Block().Height)
	if msg.BlockHeight == 0 {
		msg.BlockHeight = height
	}
	ed.stash.add(height, msg)
	return nil
}

func (ed *DefaultEventHandler) EmitBlockTx(height uint64) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	msgs, err := ed.stash.fetch(height)
	if err != nil {
		return err
	}
	ed.ethSubscriptions.Reset()
	for _, msg := range msgs {
		emitMsg, err := json.Marshal(&msg)
		if err != nil {
			log.Default.Error("Error in event marshalling for event: %v", emitMsg)
		}
		eventData := ptypes.EventData(*msg)
		ethMsg, err := proto.Marshal(&eventData)
		if err != nil {
			log.Default.Error("Error in event marshalling for event: %v", emitMsg)
		}

		log.Debug("sending event:", "height", height, "contract", msg.PluginName)
		if err := ed.dispatcher.Send(height, emitMsg); err != nil {
			log.Default.Error("Error sending event: height: %d; msg: %+v\n", height, msg)
		}
		contractTopic := "contract:" + msg.PluginName
		ed.subscriptions.Publish(pubsub.NewMessage(contractTopic, emitMsg))
		ed.ethSubscriptions.Publish(pubsub.NewMessage(string(ethMsg), emitMsg))
		for _, topic := range msg.Topics {
			ed.subscriptions.Publish(pubsub.NewMessage(topic, emitMsg))
			log.Debug("published WS event", "topic", topic)
		}
	}
	ed.stash.purge(height)
	return nil
}

// events set implementation
var exists = struct{}{}

type eventSet struct {
	m map[*EventData]struct{}
	sync.Mutex
}

func newEventSet() *eventSet {
	s := &eventSet{}
	s.m = make(map[*EventData]struct{})
	return s
}

func (s *eventSet) Add(value *EventData) {
	s.Lock()
	defer s.Unlock()
	s.m[value] = exists
}

func (s *eventSet) Remove(value *EventData) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, value)
}

func (s *eventSet) Values() []*EventData {
	s.Lock()
	defer s.Unlock()
	keys := []*EventData{}
	for k, _ := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// Set of subscription channels

type Subscription struct {
	ch        chan *EventData
	contracts []string
}

func newSubscription() *Subscription {
	return &Subscription{
		ch:        make(chan *EventData),
		contracts: make([]string, 1),
	}
}

type SubscriptionSet struct {
	pubsub.Hub
	clients map[string]pubsub.Subscriber
	sync.RWMutex
}

func NewSubscriptionSet() *SubscriptionSet {
	s := &SubscriptionSet{
		Hub:     pubsub.New(),
		clients: make(map[string]pubsub.Subscriber),
	}
	return s
}

func (s *SubscriptionSet) For(id string) (pubsub.Subscriber, bool) {
	s.Lock()
	_, exists := s.clients[id]
	if !exists {
		s.clients[id] = s.Subscribe("system:")
	}
	s.Unlock()
	return s.clients[id], exists
}

func (s *SubscriptionSet) AddSubscription(id string, topics []string) error {
	var err error
	s.Lock()
	sub, exists := s.clients[id]
	if !exists {
		err = fmt.Errorf("Subscription %s not found", id)
	} else {
		log.Debug("Adding WS subscriptions", "topics", topics)
		sub.Subscribe(append(sub.Topics(), topics...)...)
	}
	s.Unlock()
	return err
}

func (s *SubscriptionSet) Purge(id string) {
	s.Lock()
	c, _ := s.clients[id]
	s.CloseSubscriber(c)
	delete(s.clients, id)
	s.Unlock()
}

func (s *SubscriptionSet) Remove(id string, topic string) (err error) {
	s.Lock()
	c, ok := s.clients[id]
	if !ok {
		err = fmt.Errorf("Subscription not found")
	} else {
		c.Unsubscribe(topic)
		if len(c.Topics()) == 0 {
			s.Purge(id)
		}
	}
	s.Unlock()

	return err
}

// func (s *SubscriptionSet) Add(id string, contract string) (<-chan *EventData, bool) {
// 	s.Lock()
// 	defer s.Unlock()
// 	_, ok := s.m[id]
// 	exists := true
// 	if !ok {
// 		exists = false
// 		s.m[id] = newSubscription()
// 	}
// 	s.m[id].contracts = append(s.m[id].contracts, contract)
// 	return s.m[id].ch, exists
// }
//
// func (s *SubscriptionSet) Remove(id, contract string) {
// 	s.Lock()
// 	defer s.Unlock()
// 	sub, ok := s.m[id]
// 	if !ok {
// 		return
// 	}
// 	index := -1
// 	for i, c := range sub.contracts {
// 		if c == contract {
// 			index = i
// 			break
// 		}
// 	}
// 	if index < 0 {
// 		return
// 	}
// 	sub.contracts = append(sub.contracts[:index], sub.contracts[index+1:]...)
// 	if len(sub.contracts) == 0 {
// 		delete(s.m, id)
// 	}
// }
//
// func (s *SubscriptionSet) Values() []*Subscription {
// 	s.Lock()
// 	defer s.Unlock()
// 	vals := []*Subscription{}
// 	for _, v := range s.m {
// 		vals = append(vals, v)
// 	}
// 	return vals
// }

// stash is a map of height -> byteStringSet
type stash struct {
	m map[uint64]*eventSet
	sync.Mutex
}

func newStash() *stash {
	return &stash{
		m: make(map[uint64]*eventSet),
	}
}

func (s *stash) add(height uint64, msg *EventData) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.m[height]
	if !ok {
		s.m[height] = newEventSet()
	}
	s.m[height].Add(msg)
}

func (s *stash) fetch(height uint64) ([]*EventData, error) {
	s.Lock()
	defer s.Unlock()
	set, ok := s.m[height]
	if !ok {
		return nil, fmt.Errorf("stash does not exist")
	}
	return set.Values(), nil
}

func (s *stash) purge(height uint64) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, height)
}

func NewEventDispatcher(uri string) (EventDispatcher, error) {
	if strings.HasPrefix(uri, "redis") {
		return events.NewRedisEventDispatcher(uri)
	}
	return nil, fmt.Errorf("Cannot handle event dispatcher uri %s", uri)
}
