package subs

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/phonkee/go-pubsub"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

type EthSubscriptionSet struct {
	logsHub      logsResetHub
	newHeadsHub  headsResetHub
	pendingTxHub pendingTxsResetHub
}

func NewEthSubscriptionSet() *EthSubscriptionSet {
	s := &EthSubscriptionSet{
		logsHub:      *newLogsResetHubResetHub(),
		newHeadsHub:  *newHeadsResetHub(),
		pendingTxHub: *newPendingTxsResetHub(),
	}
	return s
}

func (s *EthSubscriptionSet) AddSubscription(
	method string,
	filter eth.EthFilter,
	conn *websocket.Conn) (string, error) {
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

func (s *EthSubscriptionSet) EmitEvent(data types.EventData) error {
	ethMsg, err := proto.Marshal(&data)
	if err != nil {
		return errors.Wrapf(err, "marshaling event %v", data)
	}
	s.logsHub.Publish(pubsub.NewMessage(string(ethMsg), eth.EncSubscriptionEvent(data)))
	return nil
}

func (s *EthSubscriptionSet) Reset() {
	s.logsHub.Reset()
}

func (s *EthSubscriptionSet) Remove(id string) {
	s.logsHub.closeSubscription(id)
	s.newHeadsHub.closeSubscription(id)
	s.pendingTxHub.closeSubscription(id)
}

func (s *EthSubscriptionSet) GetFilter(id string) (*eth.EthFilter, error) {
	return s.logsHub.getFilter(id)
}
