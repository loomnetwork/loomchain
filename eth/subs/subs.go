package subs

import (
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/phonkee/go-pubsub"
	abci "github.com/tendermint/abci/types"
	"sync"
)

const (
	Logs                   = "logs"
	NewHeads               = "newHeads"
	NewPendingTransactions = "newPendingTransactions"
	Syncing                = "syncing"
)

type EthSubscriptionSet struct {
	pubsub.ResetHub
	clients map[string]pubsub.Subscriber
	callers map[string][]string
	sync.RWMutex
}

func NewEthSubscriptionSet() *EthSubscriptionSet {
	s := &EthSubscriptionSet{
		ResetHub: pubsub.NewResetHub(),
		clients:  make(map[string]pubsub.Subscriber),
		callers:  make(map[string][]string),
	}
	return s
}

func (s *EthSubscriptionSet) For(caller string) (pubsub.Subscriber, string) {
	sub := s.Subscribe("system:")
	id := utils.GetId()
	s.clients[id] = sub

	s.Lock()
	s.callers[caller] = append(s.callers[caller], id)
	s.Unlock()

	return s.clients[id], id
}

func (s *EthSubscriptionSet) AddSubscription(id, method, filter string) error {
	var topics []string
	var err error
	switch method {
	case Logs:
		topics, err = topicsFromFilter(filter)
	case NewHeads:
		topics = []string{NewHeads}
	case NewPendingTransactions:
		topics = []string{NewPendingTransactions}
	case Syncing:
		err = fmt.Errorf("syncing not supported")
	default:
		err = fmt.Errorf("unrecognised method %s", method)
	}
	if err != nil {
		return err
	}

	s.Lock()
	sub, exists := s.clients[id]
	if exists {
		sub.Subscribe(append(sub.Topics(), topics...)...)
	} else {
		err = fmt.Errorf("Subscription %s not found", id)
	}
	s.Unlock()

	return err
}

func (s *EthSubscriptionSet) Purge(caller string) {
	s.Lock()
	if ids, found := s.callers[caller]; found {
		for _, id := range ids {
			if c, ok := s.clients[id]; ok {
				s.CloseSubscriber(c)
				delete(s.clients, id)
			}
		}
		delete(s.callers, caller)
	}
	s.Unlock()
}

func (s *EthSubscriptionSet) Remove(id string) (err error) {
	s.Lock()
	c, ok := s.clients[id]

	if !ok {
		err = fmt.Errorf("Subscription not found")
	} else {
		s.CloseSubscriber(c)
		delete(s.clients, id)
	}
	s.Unlock()

	return err
}

func topicsFromFilter(filter string) ([]string, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topicList := range ethFilter.Topics {
		if len(topicList) > 0 {
			for _, topic := range topicList {
				topics = append(topics, topic)
			}
		}
	}
	for _, addr := range ethFilter.Addresses {
		topics = append(topics, "contract:"+addr.String())
	}
	return topics, nil
}

func (s *EthSubscriptionSet) EmitTxEvent(data []byte, txType string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("caught panic publishing event: %v", r)
		}
	}()
	var txHash []byte
	switch txType {
	case utils.DeployEvm:
		dr := vm.DeployResponse{}
		if err := proto.Unmarshal(data, &dr); err != nil {
			return fmt.Errorf("deploy resonse does not unmarshal")
		}
		drd := vm.DeployResponseData{}
		if err := proto.Unmarshal(dr.Output, &drd); err != nil {
			return fmt.Errorf("deploy response data does not unmarshal")
		}
		txHash = drd.TxHash
	case utils.CallEVM:
		txHash = data
	default:
		return nil
	}

	result := struct {
		TxHash []byte
	}{
		TxHash: txHash,
	}
	emitMsg, _ := json.Marshal(&result)
	s.Reset()
	s.Publish(pubsub.NewMessage(NewPendingTransactions, emitMsg))
	return nil
}

func (s *EthSubscriptionSet) EmitBlockEvent(header abci.Header) (err error) {
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
		s.Reset()
		s.Publish(pubsub.NewMessage(NewHeads, emitMsg))
	}
	return nil
}
