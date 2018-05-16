package loomchain

import (
	"encoding/json"
	"fmt"
	"strings"

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
		if err := ed.dispatcher.Send(height, emitMsg); err != nil {
			log.Default.Error("Error sending event: height: %d; msg: %+v\n", height, msg)
		}
		for _, ch := range ed.subscriptions.Values() {
			ch <- msg
		}
	}
	ed.stash.purge(height)
	return nil
}

// events set implementation
var exists = struct{}{}

type eventSet struct {
	m map[*EventData]struct{}
}

func newEventSet() *eventSet {
	s := &eventSet{}
	s.m = make(map[*EventData]struct{})
	return s
}

func (s *eventSet) Add(value *EventData) {
	s.m[value] = exists
}

func (s *eventSet) Remove(value *EventData) {
	delete(s.m, value)
}

func (s *eventSet) Values() []*EventData {
	keys := []*EventData{}
	for k, _ := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// Set of subscription channels

type SubscriptionSet struct {
	m map[string]chan<- *EventData
}

func newSubscriptionSet() *SubscriptionSet {
	s := &SubscriptionSet{}
	s.m = make(map[string]chan<- *EventData)
	return s
}

func (s *SubscriptionSet) Add(id string, value chan<- *EventData) {
	s.m[id] = value
}

func (s *SubscriptionSet) Remove(id string) {
	delete(s.m, id)
}

func (s *SubscriptionSet) Values() []chan<- *EventData {
	vals := []chan<- *EventData{}
	for _, v := range s.m {
		vals = append(vals, v)
	}
	return vals
}

// stash is a map of height -> byteStringSet
type stash struct {
	m map[int64]*eventSet
}

func newStash() *stash {
	return &stash{
		m: make(map[int64]*eventSet),
	}
}

func (s *stash) add(height int64, msg *EventData) {
	_, ok := s.m[height]
	if !ok {
		s.m[height] = newEventSet()
	}
	s.m[height].Add(msg)
}

func (s *stash) fetch(height int64) ([]*EventData, error) {
	set, ok := s.m[height]
	if !ok {
		return nil, fmt.Errorf("stash does not exist")
	}
	return set.Values(), nil
}

func (s *stash) purge(height int64) {
	delete(s.m, height)
}

func NewEventDispatcher(uri string) (EventDispatcher, error) {
	if strings.HasPrefix(uri, "redis") {
		return events.NewRedisEventDispatcher(uri)
	}
	return nil, fmt.Errorf("Cannot handle event dispatcher uri %s", uri)
}
