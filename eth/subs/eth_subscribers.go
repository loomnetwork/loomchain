package subs

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
)

type topicSubscriber struct {
	wsSubscriber
	topic string
}

func newTopicSubscriber(hub pubsub.ResetHub, id, topic string, conn websocket.Conn) pubsub.Subscriber {
	wsSub := newWsSubscriber(hub, conn, id)
	return topicSubscriber{
		wsSubscriber: *wsSub,
		topic:        topic,
	}
}

func (t topicSubscriber) Match(topic string) bool {
	return t.topic == topic
}

type logSubscriber struct {
	wsSubscriber
	filter eth.EthBlockFilter
}

func newLogSubscriber(hub pubsub.ResetHub, id string, filter eth.EthFilter, conn websocket.Conn) pubsub.Subscriber {
	wsSub := newWsSubscriber(hub, conn, id)
	return logSubscriber{
		wsSubscriber: *wsSub,
		filter:       filter.EthBlockFilter,
	}
}

func (l *logSubscriber) Match(topic string) bool {
	events := types.EventData{}
	if err := proto.Unmarshal([]byte(topic), &events); err != nil {
		return false
	}

	return utils.MatchEthFilter(l.filter, events)
}
