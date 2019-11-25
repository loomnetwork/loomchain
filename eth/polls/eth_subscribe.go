// +build evm

package polls

import (
	"fmt"
	"sync"

	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain/auth"
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
	AllLogs(
		state loomchain.State,
		id string,
		readReceipts loomchain.ReadReceiptHandler,
		authCfg *auth.Config,
		createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
	) (interface{}, error)
	Poll(
		state loomchain.State,
		id string,
		readReceipts loomchain.ReadReceiptHandler,
		authCfg *auth.Config,
		createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
	) (EthPoll, interface{}, error)
	LegacyPoll(
		state loomchain.State,
		id string,
		readReceipts loomchain.ReadReceiptHandler,
		authCfg *auth.Config,
		createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
	) (EthPoll, []byte, error)
}

type EthSubscriptions struct {
	polls      map[string]EthPoll
	lastPoll   map[string]uint64
	timestamps map[uint64][]string
	mutex      sync.RWMutex // locks the 3 maps above

	lastPrune   uint64
	evmAuxStore *evmaux.EvmAuxStore
	blockStore  store.BlockStore
}

func NewEthSubscriptions(evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) *EthSubscriptions {
	p := &EthSubscriptions{
		polls:      make(map[string]EthPoll),
		lastPoll:   make(map[string]uint64),
		timestamps: make(map[uint64][]string),

		evmAuxStore: evmAuxStore,
		blockStore:  blockStore,
	}
	return p
}

func (s *EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := utils.GetId()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.polls[id] = poll
	s.lastPoll[id] = height
	s.timestamps[height] = append(s.timestamps[height], id)

	if height > BlockTimeout {
		for h := s.lastPrune; h < height-BlockTimeout; h++ {
			for _, id := range s.timestamps[h] {
				delete(s.polls, id)
				delete(s.lastPoll, id)
			}

			delete(s.timestamps, h)
		}
		s.lastPrune = height
	}

	return id
}

// This function is not thread-safe. The mutex must be locked before calling it.
func (s *EthSubscriptions) resetTimestamp(polledId string, height uint64) {
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
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	authCfg *auth.Config,
	createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
) (interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if poll, ok := s.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		return poll.AllLogs(state, id, readReceipts, authCfg, createAddressMapperCtx)
	}
}

func (s *EthSubscriptions) Poll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	authCfg *auth.Config,
	createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
) (interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	poll, ok := s.polls[id]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.Poll(state, id, readReceipts, authCfg, createAddressMapperCtx)
	s.polls[id] = newPoll

	s.resetTimestamp(id, uint64(state.Block().Height))
	return result, err

}

func (s *EthSubscriptions) LegacyPoll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	authCfg *auth.Config,
	createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
) ([]byte, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	poll, ok := s.polls[id]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.LegacyPoll(state, id, readReceipts, authCfg, createAddressMapperCtx)
	s.polls[id] = newPoll

	s.resetTimestamp(id, uint64(state.Block().Height))
	return result, err

}

func (s *EthSubscriptions) Remove(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.polls, id)
	delete(s.lastPoll, id)
}
