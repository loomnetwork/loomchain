package subs

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/rpc/eth"
	abci "github.com/tendermint/tendermint/abci/types"
)

type EthSubscriptionSet struct {
	logsHub      logsResetHub
	newHeadsHub  newHeadsResetHub
	pendingTxHub pendingTxsResetHub
}

func NewEthSubscriptionSet() *EthSubscriptionSet {
	s := &EthSubscriptionSet{
		logsHub:      *newLogsResetHubResetHub(),
		newHeadsHub:  *newNewHeadsResetHub(),
		pendingTxHub: *newPendingTxsResetHub(),
	}
	return s
}

func (s *EthSubscriptionSet) AddSubscription(method string, filter eth.EthFilter, conn websocket.Conn) (string, error) {
	var id string
	switch method {
	case Logs:
		id = s.logsHub.addSubscriber(filter, conn)
	case NewHeads:
		id = s.newHeadsHub.addSubscriber(conn)
	case NewPendingTransactions:
		id = s.pendingTxHub.addSubscriber(conn)
	case Syncing:
		return "", fmt.Errorf("syncing not supported")
	default:
		return "", fmt.Errorf("unrecognised method %s", method)
	}
	return id, nil
}

func (s *EthSubscriptionSet) EmitBlockEvent(header abci.Header) (err error) {
	return s.newHeadsHub.emitBlockEvent(header)
}

func (s *EthSubscriptionSet) EmitTxEvent(txHash []byte) (err error) {
	return s.pendingTxHub.emitTxEvent(txHash)
}
