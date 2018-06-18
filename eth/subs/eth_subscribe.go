// +build evm

package subs

import (
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/loomnetwork/loomchain"
)

var (
	BlockTimeout = uint64(10000) // blocks
)

type EthPoll interface {
	Poll(state loomchain.ReadOnlyState, id string) (EthPoll, []byte, error)
}

type EthSubscriptions struct {
	subs       map[string]EthPoll
	timestamps map[uint64][]string
}

func NewEthSubscriptions() *EthSubscriptions {
	p := &EthSubscriptions{
		subs:       make(map[string]EthPoll),
		timestamps: make(map[uint64][]string),
	}
	return p
}

func (s EthSubscriptions) Add(poll EthPoll) string {
	id := s.getId()
	s.subs[id] = poll
	return id
}

func (s EthSubscriptions) AddLogPoll(filter string) (string, error) {
	newPoll, err := NewEthLogPoll(filter)
	if err != nil {
		return "", err
	}
	return s.Add(newPoll), nil
}

func (s EthSubscriptions) AddBlockPoll(height uint64) string {
	return s.Add(NewEthBlockPoll(height))
}

func (s EthSubscriptions) AddTxPoll(height uint64) string {
	return s.Add(NewEthTxPoll(height))
}

func (s EthSubscriptions) Poll(state loomchain.ReadOnlyState, id string) ([]byte, error) {
	if poll, ok := s.subs[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.Poll(state, id)
		s.subs[id] = newPoll
		return result, err
	}
}

func (s EthSubscriptions) Remove(id string) {
	delete(s.subs, id)
}

func (s EthSubscriptions) getId() string {
	return string(rpc.NewID())
}
