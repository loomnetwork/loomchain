// +build evm

package polls

import (
	"fmt"
	"sync"

	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"

	"github.com/loomnetwork/loomchain/store"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

var (
	BlockTimeout = uint64(10 * 60) // blocks
)

type EthPoll interface {
	AllLogs(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error)
	Poll(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, interface{}, error)
	LegacyPoll(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error)
}

type EthSubscriptions struct {
	polls map[string]EthPoll
	lastPoll        map[string]uint64
	timestamps      map[uint64][]string

	pollsMutex      *sync.RWMutex
	lastPollMutex   *sync.RWMutex
	timestampsMutex *sync.RWMutex

	lastPrune       uint64
	evmAuxStore     *evmaux.EvmAuxStore
	blockStore      store.BlockStore
}

func NewEthSubscriptions(evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) *EthSubscriptions {
	p := &EthSubscriptions{
		polls:                  make(map[string]EthPoll),
		lastPoll:               make(map[string]uint64),
		timestamps:             make(map[uint64][]string),

		pollsMutex:             new(sync.RWMutex),
		lastPollMutex:          new(sync.RWMutex),
		timestampsMutex:        new(sync.RWMutex),

		evmAuxStore:            evmAuxStore,
		blockStore:             blockStore,
	}
	return p
}

func (s *EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := utils.GetId()

	s.pollsMutex.Lock()
	s.polls[id] = poll
	s.pollsMutex.Unlock()

	s.lastPollMutex.Lock()
	s.lastPoll[id] = height
	s.lastPollMutex.Unlock()

	s.timestampsMutex.Lock()
	s.timestamps[height] = append(s.timestamps[height], id)
	s.timestampsMutex.Unlock()

	s.pruneSubs(height)

	return id
}

func (s *EthSubscriptions) pruneSubs(height uint64) {
	if height > BlockTimeout {
		for h := s.lastPrune; h < height-BlockTimeout; h++ {
			s.timestampsMutex.RLock()
			for _, id := range s.timestamps[h] {
				s.Remove(id)
			}
			s.timestampsMutex.RUnlock()

			s.timestampsMutex.Lock()
			delete(s.timestamps, h)
			s.timestampsMutex.Unlock()
		}
		s.lastPrune = height
	}
}

func (s *EthSubscriptions) resetTimestamp(polledId string, height uint64) {
	s.lastPollMutex.RLock()
	lp := s.lastPoll[polledId]
	s.lastPollMutex.RUnlock()

	s.timestampsMutex.Lock()
	for i, id := range s.timestamps[lp] {
		if id == polledId {
			s.timestamps[lp] = append(s.timestamps[lp][:i], s.timestamps[lp][i+1:]...)
		}
	}
	s.timestamps[height] = append(s.timestamps[height], polledId)
	s.timestampsMutex.Unlock()

	s.lastPollMutex.Lock()
	defer s.lastPollMutex.Unlock()
	s.lastPoll[polledId] = height

}

func (s *EthSubscriptions) AddLogPoll(filter eth.EthFilter, height uint64) (string, error) {
	return s.Add(&EthLogPoll{
		filter:        filter,
		lastBlockRead: uint64(0),
		blockStore:    s.blockStore,
		evmAuxStore:   s.evmAuxStore,
	}, height), nil
}

func (s *EthSubscriptions) LegacyAddLogPoll(filter string, height uint64) (string, error) {
	newPoll, err := NewEthLogPoll(filter, s.evmAuxStore, s.blockStore)
	if err != nil {
		return "", err
	}
	return s.Add(newPoll, height), nil
}

func (s *EthSubscriptions) AddBlockPoll(height uint64) string {
	return s.Add(NewEthBlockPoll(height, s.evmAuxStore, s.blockStore), height)
}

func (s *EthSubscriptions) AddTxPoll(height uint64) string {
	return s.Add(NewEthTxPoll(height, s.evmAuxStore, s.blockStore), height)
}

func (s *EthSubscriptions) AllLogs(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	s.pollsMutex.RLock()
	defer s.pollsMutex.RUnlock()

	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		return poll.AllLogs(state, id, readReceipts)
	}
}

func (s *EthSubscriptions) Poll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	s.pollsMutex.RLock()
	poll, ok := s.polls[id]
	s.pollsMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.Poll(state, id, readReceipts)

		s.pollsMutex.Lock()
		s.polls[id] = newPoll
		s.pollsMutex.Unlock()

		s.resetTimestamp(id, uint64(state.Block().Height))
		return result, err
	}
}

func (s *EthSubscriptions) LegacyPoll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	s.pollsMutex.RLock()
	poll, ok := s.polls[id]
	s.pollsMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		newPoll, result, err := poll.LegacyPoll(state, id, readReceipts)

		s.pollsMutex.Lock()
		s.polls[id] = newPoll
		s.pollsMutex.Unlock()

		s.resetTimestamp(id, uint64(state.Block().Height))
		return result, err
	}
}

func (s *EthSubscriptions) Remove(id string) {
	s.pollsMutex.Lock()
	delete(s.polls, id)
	s.pollsMutex.Unlock()

	s.lastPollMutex.Lock()
	delete(s.lastPoll, id)
	s.lastPollMutex.Unlock()
}
