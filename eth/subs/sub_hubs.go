package subs

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	Logs                   = "logs"
	NewHeads               = "newHeads"
	NewPendingTransactions = "newPendingTransactions"
	Syncing                = "syncing"
)

type headsResetHub struct {
	ethResetHub
}

func newHeadsResetHub() *headsResetHub {
	hub := newEthResetHub()
	return &headsResetHub{
		ethResetHub: *hub,
	}
}

func (pt *headsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(pt, id, NewHeads, conn)
	pt.clients[id] = sub
	pt.unsent[id] = true
	return id
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash and
// https://github.com/ethereum/go-ethereum/wiki/RPC-PUB-SUB
// both suggest we should not show the block's hash and details of the blocks transactions
// however we could do, as the information is available at this point.
func (nh *headsResetHub) emitBlockEvent(header abci.Header) (err error) {
	if len(nh.clients) > 0 {
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
	}
	return nil
}

type pendingTxsResetHub struct {
	ethResetHub
}

func newPendingTxsResetHub() *pendingTxsResetHub {
	hub := newEthResetHub()
	return &pendingTxsResetHub{
		ethResetHub: *hub,
	}
}

func (pt *pendingTxsResetHub) addSubscriber(conn websocket.Conn) string {
	id := utils.GetId()
	sub := newTopicSubscriber(pt, id, NewPendingTransactions, conn)
	pt.clients[id] = sub
	pt.unsent[id] = true
	return id
}

func (pt *pendingTxsResetHub) emitTxEvent(txHash []byte) (err error) {
	if len(pt.clients) > 0 {
		txHashRawJson, err := json.Marshal(hex.EncodeToString(txHash))
		if err != nil {
			return errors.Wrapf(err, "json marshaling tx hash %v", txHash)
		}
		pt.Reset()
		pt.Publish(pubsub.NewMessage(NewPendingTransactions, txHashRawJson))
	}
	return nil
}

type logsResetHub struct {
	ethResetHub
	lMutex      *sync.RWMutex
}

func newLogsResetHubResetHub() *logsResetHub {
	hub := newEthResetHub()
	return &logsResetHub{
		ethResetHub: *hub,
		lMutex:      &sync.RWMutex{},
	}
}

func (l *logsResetHub) getFilter(id string) (*eth.EthFilter, error) {
	l.lMutex.RLock()
	defer l.lMutex.RUnlock()
	if _,ok := l.clients[id]; !ok {
		return nil, fmt.Errorf("finding subscriber for id %s", id)
	}

	if filter, ok := l.clients[id].(logSubscriber); ok {
		return &eth.EthFilter{ EthBlockFilter: filter.filter }, nil
	} else {
		panic("clients can only be logSubscribers")
	}
}

func (l *logsResetHub) addSubscriber(filter eth.EthFilter, conn websocket.Conn) string {
	id := utils.GetId()
	sub := newLogSubscriber(l, id, filter, conn)
	l.clients[id] = sub
	l.unsent[id] = true
	return id
}
