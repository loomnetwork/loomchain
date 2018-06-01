package loomchain

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain/abci/backend"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/log"
)

type EventData struct {
	Caller      loom.Address `json:"caller"`
	Address     loom.Address `json:"address"`
	PluginName  string       `json:"plugin"`
	BlockHeight int64        `json:"blockHeight"`
	Data        []byte       `json:"encodedData"`
	RawRequest  []byte       `json:"rawRequest"`
}

func (e *EventData) AssertIsTMEventData() {}

type EventHandler interface {
	Post(state State, e *EventData) error
	EmitBlockTx(height int64) error
	SubscriptionSet() *SubscriptionSet
}

type EventDispatcher interface {
	Send(index int64, msg []byte) error
}

type DefaultEventHandler struct {
	dispatcher    EventDispatcher
	stash         *stash
	backend       backend.Backend
	subscriptions *SubscriptionSet
}

func NewDefaultEventHandler(dispatcher EventDispatcher, b backend.Backend) *DefaultEventHandler {
	return &DefaultEventHandler{
		dispatcher:    dispatcher,
		stash:         newStash(),
		backend:       b,
		subscriptions: newSubscriptionSet(),
	}
}

func (ed *DefaultEventHandler) SubscriptionSet() *SubscriptionSet {
	return ed.subscriptions
}

func (ed *DefaultEventHandler) Post(state State, msg *EventData) error {
	height := state.Block().Height
	if msg.BlockHeight == 0 {
		msg.BlockHeight = height
	}
	ed.stash.add(height, msg)
	return nil
}

func (ed *DefaultEventHandler) EmitBlockTx(height int64) error {
	msgs, err := ed.stash.fetch(height)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		emitMsg, err := json.Marshal(&msg)
		if err != nil {
			log.Default.Error("Error in event marshalling for event: %v", emitMsg)
		}
		log.Debug("sending event: height: %d; msg: %+v\n", height, msg)
		if err := ed.dispatcher.Send(height, emitMsg); err != nil {
			log.Default.Error("Error sending event: height: %d; msg: %+v\n", height, msg)
		}
		for _, sub := range ed.subscriptions.Values() {
			var match bool
			if len(sub.contracts) > 0 { // empty filter means match anything
				for _, c := range sub.contracts {
					if msg.PluginName == c {
						match = true
						break
					}
				}
			} else {
				match = true
			}
			if match == true {
				sub.ch <- msg
			}
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
	m map[string]*Subscription
	sync.Mutex
}

func newSubscriptionSet() *SubscriptionSet {
	s := &SubscriptionSet{}
	s.m = make(map[string]*Subscription)
	return s
}

func (s *SubscriptionSet) Add(id string, contract string) (<-chan *EventData, bool) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.m[id]
	exists := true
	if !ok {
		exists = false
		s.m[id] = newSubscription()
	}
	s.m[id].contracts = append(s.m[id].contracts, contract)
	return s.m[id].ch, exists
}

func (s *SubscriptionSet) Remove(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, id)
}

func (s *SubscriptionSet) Values() []*Subscription {
	s.Lock()
	defer s.Unlock()
	vals := []*Subscription{}
	for _, v := range s.m {
		vals = append(vals, v)
	}
	return vals
}

// stash is a map of height -> byteStringSet
type stash struct {
	m map[int64]*eventSet
	sync.Mutex
}

func newStash() *stash {
	return &stash{
		m: make(map[int64]*eventSet),
	}
}

func (s *stash) add(height int64, msg *EventData) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.m[height]
	if !ok {
		s.m[height] = newEventSet()
	}
	s.m[height].Add(msg)
}

func (s *stash) fetch(height int64) ([]*EventData, error) {
	s.Lock()
	defer s.Unlock()
	set, ok := s.m[height]
	if !ok {
		return nil, fmt.Errorf("stash does not exist")
	}
	return set.Values(), nil
}

func (s *stash) purge(height int64) {
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
