package subs

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
	abci "github.com/tendermint/tendermint/abci/types"
	"sync"
)

const (
	Logs                   = "logs"
	NewHeads               = "newHeads"
	NewPendingTransactions = "newPendingTransactions"
	Syncing                = "syncing"
)

type newHeadsResetHub struct {
	ethResetHub
	clients     map[string]pubsub.Subscriber
	nhMutex     *sync.RWMutex
}

func newNewHeadsResetHub() *newHeadsResetHub {
	hub := newEthResetHub()
	return &newHeadsResetHub{
		ethResetHub: *hub,
		clients:     make(map[string]pubsub.Subscriber),
		nhMutex:     &sync.RWMutex{},
	}
}

func (nh *newHeadsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(nh, id, NewHeads, conn)
	nh.clients[id] = sub
	return id
}

func (h *newHeadsResetHub) closeSubscription(id string) {
	h.nhMutex.Lock()
	if sub, ok := h.clients[id]; ok {
		delete(h.clients, id)
		h.CloseSubscriber(sub)
	}
	h.nhMutex.Unlock()
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
	ethResetHub
	clients     map[string]pubsub.Subscriber
	ptMutex     *sync.RWMutex
}

func newPendingTxsResetHub() *pendingTxsResetHub {
	hub := newEthResetHub()
	return &pendingTxsResetHub{
		ethResetHub: *hub,
		clients:     make(map[string]pubsub.Subscriber),
		ptMutex:     &sync.RWMutex{},
	}
}

func (pt *pendingTxsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(pt, id, NewPendingTransactions, conn)
	pt.clients[id] = sub
	return id
}

func (pt *pendingTxsResetHub) closeSubscription(id string) {
	pt.ptMutex.Lock()
	if sub, ok := pt.clients[id]; ok {
		delete(pt.clients, id)
		pt.CloseSubscriber(sub)
	}
	pt.ptMutex.Unlock()
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
	ethResetHub
	clients     map[string]logSubscriber
	lMutex      *sync.RWMutex
}

func newLogsResetHubResetHub() *logsResetHub {
	hub := newEthResetHub()
	return &logsResetHub{
		ethResetHub: *hub,
		clients:     make(map[string]logSubscriber),
		lMutex:      &sync.RWMutex{},
	}
}

func (l *logsResetHub) closeSubscrilion(id string) {
	l.lMutex.Lock()
	if sub, ok := l.clients[id]; ok {
		delete(l.clients, id)
		l.CloseSubscriber(&sub)
	}
	l.lMutex.Unlock()
}

func (l *logsResetHub) addSubscriber(filter eth.EthFilter, conn websocket.Conn) string {
	id := utils.GetId()
	sub := newLogSubscriber(l, id, filter, conn)
	l.clients[id] = sub
	return id
}
