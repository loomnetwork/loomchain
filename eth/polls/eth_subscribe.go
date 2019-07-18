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
	mutex      *sync.RWMutex
	lastPrune       uint64
	evmAuxStore     *evmaux.EvmAuxStore
	blockStore      store.BlockStore
}

func NewEthSubscriptions(evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) *EthSubscriptions {
	p := &EthSubscriptions{
		polls:                  make(map[string]EthPoll),
		lastPoll:               make(map[string]uint64),
		timestamps:             make(map[uint64][]string),
		mutex:             new(sync.RWMutex),

		evmAuxStore:            evmAuxStore,
		blockStore:             blockStore,
	}
	return p
}

func (s *EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := utils.GetId()

	s.mutex.Lock()
	s.polls[id] = poll
	s.lastPoll[id] = height
	s.timestamps[height] = append(s.timestamps[height], id)
	s.mutex.Unlock()

	s.pruneSubs(height)

	return id
}

func (s *EthSubscriptions) pruneSubs(height uint64) {
	if height > BlockTimeout {
		for h := s.lastPrune; h < height-BlockTimeout; h++ {
			idsToRemove := []string{}

			s.mutex.RLock()
			for _, id := range s.timestamps[h] {
				idsToRemove = append(idsToRemove, id)
			}
			s.mutex.RUnlock()

			for _, id := range idsToRemove {
				s.Remove(id)
			}

			s.mutex.Lock()
			delete(s.timestamps, h)
			s.mutex.Unlock()
		}
		s.lastPrune = height
	}
}

func (s *EthSubscriptions) resetTimestamp(polledId string, height uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	lp := s.lastPoll[polledId]
	for i, id := range s.timestamps[lp] {
		if id == polledId {
			s.timestamps[lp] = append(s.timestamps[lp][:i], s.timestamps[lp][i+1:]...)
		}
	}
	s.timestamps[height] = append(s.timestamps[height], polledId)
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
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		return poll.AllLogs(state, id, readReceipts)
	}
}

func (s *EthSubscriptions) Poll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	s.mutex.Lock()
	poll, ok := s.polls[id]
	if !ok {
		s.mutex.Unlock()
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.Poll(state, id, readReceipts)
	s.polls[id] = newPoll
	s.mutex.Unlock()

	s.resetTimestamp(id, uint64(state.Block().Height))
	return result, err

}

func (s *EthSubscriptions) LegacyPoll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	s.mutex.Lock()
	poll, ok := s.polls[id]
	if !ok {
		s.mutex.Unlock()
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.LegacyPoll(state, id, readReceipts)
	s.polls[id] = newPoll
	s.mutex.Unlock()

	s.resetTimestamp(id, uint64(state.Block().Height))
	return result, err

}

func (s *EthSubscriptions) Remove(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.polls, id)
	delete(s.lastPoll, id)
}
