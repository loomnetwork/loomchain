package subs

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	Logs                   = "logs"
	NewHeads               = "newHeads"
	NewPendingTransactions = "newPendingTransactions"
	Syncing                = "syncing"
)

type newHeadsResetHub struct {
	pubsub.ResetHub
	//maps id to subscriber
	clients map[string]pubsub.Subscriber
}

func newNewHeadsResetHub() *newHeadsResetHub {
	hub := newEthResetHub()
	return &newHeadsResetHub{
		ResetHub: hub,
		clients:  make(map[string]pubsub.Subscriber),
	}
}

func (nh *newHeadsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(nh, id, NewHeads, conn)
	nh.clients[id] = sub
	return id
}

func (nh *newHeadsResetHub) emitBlockEvent(header abci.Header) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	blockinfo := types.EthBlockInfo{
		ParentHash: header.LastBlockHash,
		Number:     header.Height,
		Timestamp:  header.Time,
	}
	emitMsg, err := json.Marshal(&blockinfo)
	if err == nil {
		nh.Reset()
		nh.Publish(pubsub.NewMessage(NewHeads, emitMsg))
	}
	return nil
}

type pendingTxsResetHub struct {
	pubsub.ResetHub
	//maps id to subscriber
	clients map[string]pubsub.Subscriber
}

func newPendingTxsResetHub() *pendingTxsResetHub {
	hub := newEthResetHub()
	return &pendingTxsResetHub{
		ResetHub: hub,
		clients:  make(map[string]pubsub.Subscriber),
	}
}

func (pt *pendingTxsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(pt, id, NewPendingTransactions, conn)
	pt.clients[id] = sub
	return id
}

func (pt *pendingTxsResetHub) emitTxEvent(txHash []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	result := struct {
		TxHash []byte
	}{
		TxHash: txHash,
	}
	emitMsg, _ := json.Marshal(&result)
	pt.Reset()
	pt.Publish(pubsub.NewMessage(NewPendingTransactions, emitMsg))
	return nil
}

type logsResetHub struct {
	pubsub.ResetHub
	//maps id to subscriber
	clients map[string]pubsub.Subscriber
}

func newLogsResetHubResetHub() *logsResetHub {
	hub := newEthResetHub()
	return &logsResetHub{
		ResetHub: hub,
		clients:  make(map[string]pubsub.Subscriber),
	}
}

func (l *logsResetHub) addSubscriber(filter eth.EthFilter, conn websocket.Conn) string {
	id := utils.GetId()
	sub := newLogSubscriber(l, id, filter, conn)
	l.clients[id] = sub
	return id
}
