// +build evm

package polls

import (
	"fmt"

	"github.com/loomnetwork/loomchain/store"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

var (
	BlockTimeout = uint64(10 * 60) // blocks
)

type EthPoll interface {
	AllLogs(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
		readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) (interface{}, error)
	Poll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
		readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) (EthPoll, interface{}, error)
	LegacyPoll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
		readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) (EthPoll, []byte, error)
}

type EthSubscriptions struct {
	polls map[string]EthPoll

	lastPoll   map[string]uint64
	timestamps map[uint64][]string
	lastPrune  uint64
}

func NewEthSubscriptions() *EthSubscriptions {
	p := &EthSubscriptions{
		polls:      make(map[string]EthPoll),
		lastPoll:   make(map[string]uint64),
		timestamps: make(map[uint64][]string),
	}
	return p
}

func (s EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := utils.GetId()
	s.polls[id] = poll

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

func (s EthSubscriptions) AddLogPoll(filter eth.EthFilter, height uint64) (string, error) {
	return s.Add(&EthLogPoll{
		filter:        filter,
		lastBlockRead: uint64(0),
	}, height), nil
}

func (s EthSubscriptions) LegacyAddLogPoll(filter string, height uint64) (string, error) {
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

func (s EthSubscriptions) AllLogs(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) (interface{}, error) {
	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		return poll.AllLogs(blockStore, state, id, readReceipts, evmAuxStore)
	}
}

func (s EthSubscriptions) Poll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) (interface{}, error) {
	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.Poll(blockStore, state, id, readReceipts, evmAuxStore)
		s.polls[id] = newPoll
		s.resetTimestamp(id, uint64(state.Block().Height))
		return result, err
	}
}

func (s EthSubscriptions) LegacyPoll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *store.EvmAuxStore) ([]byte, error) {
	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.LegacyPoll(blockStore, state, id, readReceipts, evmAuxStore)
		s.polls[id] = newPoll
		s.resetTimestamp(id, uint64(state.Block().Height))
		return result, err
	}
}

func (s EthSubscriptions) Remove(id string) {
	delete(s.polls, id)
	delete(s.lastPoll, id)
}
