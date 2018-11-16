package subs

import (
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/phonkee/go-pubsub"
)

const (
	Logs                   = "logs"
	NewHeads               = "newHeads"
	NewPendingTransactions = "newPendingTransactions"
	Syncing                = "syncing"
)

//newTopicSubscriber(hub pubsub.ResetHub, id, topic string, conn websocket.Conn) pubsub.Subscriber

type WSTopicRestHub struct {
	pubsub.ResetHub
	topic string
	//maps id to subscriber
	clients map[string]pubsub.Subscriber
}

func newWSTopicResetHub(topic string) *WSTopicRestHub {
	hub := newEthResetHub()
	return &WSTopicRestHub{
		ResetHub: hub,
		topic:    topic,
	}
}

func (t *WSTopicRestHub) AddSubscriber(conn websocket.Conn) (string, pubsub.Subscriber) {
	id := utils.GetId()
	sub := newTopicSubscriber(t.ResetHub, id, t.topic, conn)
	t.clients[id] = sub
	return id, sub
}
