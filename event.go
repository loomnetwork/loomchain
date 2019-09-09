package loomchain

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/log"
	pubsub "github.com/phonkee/go-pubsub"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type EventData types.EventData

// EventHandler handles collection, storage, and dispatch of tx & block events.
type EventHandler interface {
	// Post stores an event in transient storage, it may be emitted later by EmitBlockTx.
	Post(height uint64, e *types.EventData) error
	// Commit adds events posted since the last Rollback to the set of events to be emitted when
	// EmitBlockTx is called.
	Commit(height uint64)
	// Rollback discards any posted events that haven't been committed.
	Rollback()
	// Emits all events committed while processing the specified block.
	EmitBlockTx(height uint64, blockTime time.Time) error
	SubscriptionSet() *SubscriptionSet
	EthSubscriptionSet() *subs.EthSubscriptionSet
	LegacyEthSubscriptionSet() *subs.LegacyEthSubscriptionSet
}

type EventDispatcher interface {
	Send(blockHeight uint64, eventIndex int, msg []byte) error
	Flush()
}

type DefaultEventHandler struct {
	dispatcher             EventDispatcher
	stash                  *stash
	eventCache             []*EventData
	subscriptions          *SubscriptionSet
	ethSubscriptions       *subs.EthSubscriptionSet
	legacyEthSubscriptions *subs.LegacyEthSubscriptionSet
}

func NewDefaultEventHandler(dispatcher EventDispatcher) *DefaultEventHandler {
	return &DefaultEventHandler{
		dispatcher:             dispatcher,
		stash:                  newStash(),
		subscriptions:          NewSubscriptionSet(),
		ethSubscriptions:       subs.NewEthSubscriptionSet(),
		legacyEthSubscriptions: subs.NewLegacyEthSubscriptionSet(),
	}
}

func (ed *DefaultEventHandler) SubscriptionSet() *SubscriptionSet {
	return ed.subscriptions
}

func (ed *DefaultEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return ed.ethSubscriptions
}

func (ed *DefaultEventHandler) LegacyEthSubscriptionSet() *subs.LegacyEthSubscriptionSet {
	return ed.legacyEthSubscriptions
}

func (ed *DefaultEventHandler) Post(height uint64, msg *types.EventData) error {
	if msg.BlockHeight == 0 {
		msg.BlockHeight = height
	}
	// TODO: this is stupid, fix it
	eventData := EventData(*msg)
	ed.eventCache = append(ed.eventCache, &eventData)
	return nil
}

func (ed *DefaultEventHandler) Commit(height uint64) {
	ed.stash.add(height, ed.eventCache)
	ed.eventCache = nil
}

func (ed *DefaultEventHandler) Rollback() {
	ed.eventCache = nil
}

func (ed *DefaultEventHandler) EmitBlockTx(height uint64, blockTime time.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	msgs, err := ed.stash.fetch(height)
	if err != nil {
		return err
	}

	ed.legacyEthSubscriptions.Reset()
	ed.ethSubscriptions.Reset()
	// Timestamp added here rather than being stored in the event itself so
	// as to avoid altering the data saved to the app-store.
	timestamp := blockTime.Unix()

	for i, msg := range msgs {
		msg.BlockTime = timestamp
		emitMsg, err := json.Marshal(&msg)
		if err != nil {
			log.Default.Error("Error in event marshalling for event", "message", emitMsg)
		}
		eventData := types.EventData(*msg)
		ethMsg, err := proto.Marshal(&eventData)
		if err != nil {
			log.Default.Error("Error in event marshalling for event", "message", emitMsg)
		}

		log.Debug("sending event:", "height", height, "contract", msg.PluginName)
		if err := ed.dispatcher.Send(height, i, emitMsg); err != nil {
			log.Default.Error("Failed to dispatch event", "err", err, "height", height, "msg", msg)
		}
		contractTopic := "contract:" + msg.PluginName
		ed.subscriptions.Publish(pubsub.NewMessage(contractTopic, emitMsg))
		if err := ed.ethSubscriptions.EmitEvent(eventData); err != nil {
			log.Error("Failed to emit subscription event", "err", err, "height", height, "msg", msg)
		}
		ed.legacyEthSubscriptions.Publish(pubsub.NewMessage(string(ethMsg), emitMsg))
		for _, topic := range msg.Topics {
			ed.subscriptions.Publish(pubsub.NewMessage(topic, emitMsg))
			log.Debug("published WS event", "topic", topic)
		}
	}
	ed.dispatcher.Flush()
	ed.stash.purge(height)
	return nil
}

// InstrumentingEventHandler captures metrics and implements EventHandler
type InstrumentingEventHandler struct {
	methodDuration metrics.Histogram
	next           EventHandler
}

var _ EventHandler = &InstrumentingEventHandler{}

// NewInstrumentingEventHandler initializes the metrics and maintains event handler
func NewInstrumentingEventHandler(handler EventHandler) EventHandler {
	// initialize metrics
	fieldKeys := []string{"method", "error"}
	methodDuration := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "event_handler",
		Name:       "method_duration",
		Help:       "Total duration of requests in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)

	return &InstrumentingEventHandler{
		methodDuration: methodDuration,
		next:           handler,
	}
}

// Post captures the metrics
func (m InstrumentingEventHandler) Post(height uint64, e *types.EventData) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Post", "error", fmt.Sprint(err != nil)}
		m.methodDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = m.next.Post(height, e)
	return
}

func (m InstrumentingEventHandler) Commit(height uint64) {
	m.next.Commit(height)
}

func (m InstrumentingEventHandler) Rollback() {
	m.next.Rollback()
}

// EmitBlockTx captures the metrics
func (m InstrumentingEventHandler) EmitBlockTx(height uint64, blockTime time.Time) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EmitBlockTx", "error", fmt.Sprint(err != nil)}
		m.methodDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = m.next.EmitBlockTx(height, blockTime)
	return
}

func (m InstrumentingEventHandler) SubscriptionSet() *SubscriptionSet {
	return m.next.SubscriptionSet()
}

func (m InstrumentingEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return m.next.EthSubscriptionSet()
}

func (m InstrumentingEventHandler) LegacyEthSubscriptionSet() *subs.LegacyEthSubscriptionSet {
	return m.next.LegacyEthSubscriptionSet()
}

// TODO: remove? It's just a wrapper of []*EventData
// events set implementation
type eventSet struct {
	events []*EventData
}

func newEventSet() *eventSet {
	s := &eventSet{}
	s.events = []*EventData{}
	return s
}

func (s *eventSet) Add(value *EventData) {
	s.events = append(s.events, value)
}

func (s *eventSet) Values() []*EventData {
	return s.events
}

// Set of subscription channels
type SubscriptionSet struct {
	pubsub.Hub
	// maps ID (remote socket address) to subscriber
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

// For returns a subscriber matching the given ID, creating a new one if needed.
// New subscribers are subscribed to a single "system:" topic.
// Returns true if the subscriber already existed, and false if a new one was created.
func (s *SubscriptionSet) For(id string) (pubsub.Subscriber, bool) {
	s.Lock()
	_, exists := s.clients[id]
	if !exists {
		s.clients[id] = s.Subscribe("system:")
	}
	res := s.clients[id]
	s.Unlock()
	return res, exists
}

// AddSubscription subscribes the subscriber matching the given ID to additional topics (existing
// topics are retained).
// An error will be returned if a subscriber matching the given ID doesn't exist.
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
	c := s.clients[id]
	s.CloseSubscriber(c)
	delete(s.clients, id)
	s.Unlock()
}

// Remove unsubscribes a subscriber from the specified topic, if this is the only topic the subscriber
// was subscribed to then the subscriber is removed from the set.
// An error will be returned if a subscriber matching the given ID doesn't exist.
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

func (s *stash) add(height uint64, msgs []*EventData) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.m[height]
	if !ok {
		s.m[height] = newEventSet()
	}
	for _, msg := range msgs {
		s.m[height].Add(msg)
	}
}

func (s *stash) fetch(height uint64) ([]*EventData, error) {
	s.Lock()
	defer s.Unlock()
	set, ok := s.m[height]
	if !ok {
		return nil, nil
	}
	return set.Values(), nil
}

func (s *stash) purge(height uint64) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, height)
}
