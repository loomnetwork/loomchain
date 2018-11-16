package subs

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
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

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash and
// https://github.com/ethereum/go-ethereum/wiki/RPC-PUB-SUB
// both suggest we should not show the block's hash and details of the blocks transactions
// however we could do, as the information is available at this point.
func (nh *newHeadsResetHub) emitBlockEvent(header abci.Header) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	blockinfo := eth.JsonBlockObject{
		ParentHash: eth.EncBytes(header.LastBlockHash),
		Number:     eth.EncInt(header.Height),
		Timestamp:  eth.EncInt(header.Time),
		GasLimit:   eth.EncInt(0),
		GasUsed:    eth.EncInt(0),
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
	pt.Reset()
	pt.Publish(pubsub.NewMessage(NewPendingTransactions, txHash))
	return nil
}

type logsResetHub struct {
	pubsub.ResetHub
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
