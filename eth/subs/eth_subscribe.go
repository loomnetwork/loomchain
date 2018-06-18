// +build evm

package subs

import (
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/loomnetwork/loomchain"
)

var (
	BlockTimeout = uint64(10 * 60) // blocks
)

type EthPoll interface {
	Poll(state loomchain.ReadOnlyState, id string) (EthPoll, []byte, error)
}

type EthSubscriptions struct {
	subs map[string]EthPoll

	lastPoll   map[string]uint64
	timestamps map[uint64][]string
	lastPrune  uint64
}

func NewEthSubscriptions() *EthSubscriptions {
	p := &EthSubscriptions{
		subs:       make(map[string]EthPoll),
		lastPoll:   make(map[string]uint64),
		timestamps: make(map[uint64][]string),
	}
	return p
}

func (s EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := s.getId()
	s.subs[id] = poll

	s.lastPoll[id] = height
	s.timestamps[height] = append(s.timestamps[height], id)
	s.pruneSubs(height)

	return id
}

func (s EthSubscriptions) pruneSubs(height uint64) {
	if height > BlockTimeout {
		for h := s.lastPrune; h < height-BlockTimeout; h++ {
			for _, id := range s.timestamps[h] {
				s.Remove(id)
			}
			delete(s.timestamps, h)
		}
		s.lastPrune = height
	}
}

func (s EthSubscriptions) resetTimestamp(polledId string, height uint64) {
	lp := s.lastPoll[polledId]
	for i, id := range s.timestamps[lp] {
		if id == polledId {
			s.timestamps[lp] = append(s.timestamps[lp][:i], s.timestamps[lp][i+1:]...)
		}
	}
	s.lastPoll[polledId] = height
	s.timestamps[height] = append(s.timestamps[height], polledId)
}

func (s EthSubscriptions) AddLogPoll(filter string, height uint64) (string, error) {
	newPoll, err := NewEthLogPoll(filter)
	if err != nil {
		return "", err
	}
	return s.Add(newPoll, height), nil
}

func (s EthSubscriptions) AddBlockPoll(height uint64) string {
	return s.Add(NewEthBlockPoll(height), height)
}

func (s EthSubscriptions) AddTxPoll(height uint64) string {
	return s.Add(NewEthTxPoll(height), height)
}

func (s EthSubscriptions) Poll(state loomchain.ReadOnlyState, id string) ([]byte, error) {
	if poll, ok := s.subs[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.Poll(state, id)
		s.subs[id] = newPoll
		s.resetTimestamp(id, uint64(state.Block().Height))
		return result, err
	}
}

func (s EthSubscriptions) Remove(id string) {
	delete(s.subs, id)
	delete(s.lastPoll, id)
}

func (s EthSubscriptions) getId() string {
	return string(rpc.NewID())
}
