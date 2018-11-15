package subs

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/log"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
)

type EthWSJsonResult struct {
	Result       json.RawMessage `json:"result"`
	Subscription string          `json:"subscription"`
}

type EthWSJsonRpcResponse struct {
	Params  EthWSJsonResult `json:"params"`
	Version string          `json:"jsonrpc"`
	Method  string          `json:"id"`
}

// ethDepreciatedSubscriber is Subscriber implementation
type ethSubscriber struct {
	hub    pubsub.ResetHub
	mutex  *sync.RWMutex
	sf     pubsub.SubscriberFunc
	filter eth.EthBlockFilter
	id     string
}

// Close ethDepreciatedSubscriber removes ethDepreciatedSubscriber from hub and stops receiving messages
func (s *ethSubscriber) Close() {
	s.hub.CloseSubscriber(s)
}

// Do sets ethDepreciatedSubscriber function that will be called when message arrives
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

	return utils.MatchEthFilter(s.filter, events)
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

type ethWSSubscriber struct {
	ethSubscriber
	conn websocket.Conn
}

func newWSEthSubscriber(hub pubsub.ResetHub, filter eth.EthFilter, conn websocket.Conn, id string) pubsub.Subscriber {
	sf := func(msg pubsub.Message) {
		resp := EthWSJsonRpcResponse{
			Params:  EthWSJsonResult{msg.Body(), id},
			Version: "2.0",
			Method:  "eth_subscription",
		}

		jsonBytes, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			log.Error("error %v marshalling event %v, id %s", err, msg, id)
		}
		if err := conn.WriteMessage(websocket.TextMessage, jsonBytes); err != nil {
			log.Error("error %v writing event %v to websocket, id %s ", err, jsonBytes, id)
		}
	}

	return &ethWSSubscriber{
		ethSubscriber: ethSubscriber{
			hub:    hub,
			mutex:  &sync.RWMutex{},
			filter: filter.EthBlockFilter,
			id:     id,
			sf:     sf,
		},
		conn: conn,
	}
}

func (s *ethWSSubscriber) Publish(message pubsub.Message) int {
	if s.sf == nil {
		return 0
	}
	ethMsg := types.EthMessage{
		Body: message.Body(),
		Id:   s.id,
	}
	msg, err := proto.Marshal(&ethMsg)
	if err != nil {
		return 0
	}
	s.sf(pubsub.NewMessage(message.Topic(), msg))
	return 1
}

func (s *ethWSSubscriber) Subscribe(topics ...string) pubsub.Subscriber {
	var topic []byte
	if len(topics) > 0 {
		topic = []byte(topics[0])
	} else {
		topic = []byte{}
	}
	filter, err := utils.UnmarshalEthFilter(topic)
	if err == nil {
		s.filter = filter.EthBlockFilter
	}

	return s
}

// newSubscriber returns ethDepreciatedSubscriber for given topics
func newEthDepreciatedSubscriber(hub pubsub.ResetHub, topics ...string) (result pubsub.Subscriber) {
	f, err := utils.UnmarshalEthFilter([]byte(topics[0]))
	var filter eth.EthBlockFilter
	if err == nil {
		filter = f.EthBlockFilter
	}
	result = &ethSubscriber{
		ethSubscriber: ethSubscriber{
			hub:    hub,
			mutex:  &sync.RWMutex{},
			sf:     nil,
			filter: filter,
		},
	}
	return result
}

// Publish publishes message to ethDepreciatedSubscriber
func (s *ethSubscriber) Publish(message pubsub.Message) int {
	if s.sf == nil {
		return 0
	}
	ethMsg := types.EthMessage{
		Body: message.Body(),
		Id:   s.id,
	}
	msg, err := proto.Marshal(&ethMsg)
	if err != nil {
		return 0
	}
	s.sf(pubsub.NewMessage(message.Topic(), msg))
	return 1
}

// Subscribe subscribes to topics
func (s *ethSubscriber) Subscribe(topics ...string) pubsub.Subscriber {
	var topic []byte
	if len(topics) > 0 {
		topic = []byte(topics[0])
	} else {
		topic = []byte{}
	}
	filter, err := utils.UnmarshalEthFilter(topic)
	if err == nil {
		s.filter = filter.EthBlockFilter
	}

	return s
}
