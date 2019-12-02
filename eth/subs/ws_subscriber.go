package subs

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/log"
	"github.com/phonkee/go-pubsub"
	"sync"
)

type ethWSJsonResult struct {
	Result       json.RawMessage `json:"result"`
	Subscription string          `json:"subscription"`
}

type ethWSJsonRpcResponse struct {
	Params  ethWSJsonResult `json:"params"`
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
}

type wsSubscriber struct {
	hub   pubsub.ResetHub
	mutex *sync.RWMutex
	sf    pubsub.SubscriberFunc
	id    string
	conn  *websocket.Conn
}

func newWsSubscriber(hub pubsub.ResetHub, conn *websocket.Conn, id string) *wsSubscriber {
	sf := func(msg pubsub.Message) {
		resp := ethWSJsonRpcResponse{
			Params:  ethWSJsonResult{msg.Body(), id},
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

	return &wsSubscriber{
		hub:   hub,
		mutex: &sync.RWMutex{},
		id:    id,
		sf:    sf,
		conn:  conn,
	}
}

// Do nothing. Closing websocket connection is done by the handler not here.
func (s wsSubscriber) Close() {
}

// Do sets subscriber function that will be called when message arrives
func (s wsSubscriber) Do(sf pubsub.SubscriberFunc) pubsub.Subscriber {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sf = sf
	return s
}

// Match returns whether subscriber topics matches
func (s wsSubscriber) Match(topic string) bool {
	return false
}

// Publish publishes message to subscriber
func (s wsSubscriber) Publish(message pubsub.Message) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.sf == nil {
		return 0
	}
	s.sf(message)

	return 1
}

// Subscribe subscribes to topics
func (s wsSubscriber) Subscribe(topics ...string) pubsub.Subscriber {
	return s
}

// Topics returns whole list of all topics subscribed to
func (s wsSubscriber) Topics() []string {
	return []string{}
}

// Unsubscribe unsubscribes from given topics (exact match)
func (s wsSubscriber) Unsubscribe(topics ...string) pubsub.Subscriber {
	return s
}
