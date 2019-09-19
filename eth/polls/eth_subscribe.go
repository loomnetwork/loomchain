// +build evm

package polls

import (
	"fmt"
	"sync"

	"github.com/loomnetwork/loomchain/state"
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
	AllLogs(s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error)
	Poll(s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, interface{}, error)
	LegacyPoll(s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error)
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

func (es *EthSubscriptions) Add(poll EthPoll, height uint64) string {
	id := utils.GetId()

	es.mutex.Lock()
	defer es.mutex.Unlock()

	es.polls[id] = poll
	es.lastPoll[id] = height
	es.timestamps[height] = append(es.timestamps[height], id)

	if height > BlockTimeout {
		for h := es.lastPrune; h < height-BlockTimeout; h++ {
			for _, id := range es.timestamps[h] {
				delete(es.polls, id)
				delete(es.lastPoll, id)
			}

			delete(es.timestamps, h)
		}
		es.lastPrune = height
	}

	return id
}

// This function is not thread-safe. The mutex must be locked before calling it.
func (es *EthSubscriptions) resetTimestamp(polledId string, height uint64) {
	lp := es.lastPoll[polledId]
	for i, id := range es.timestamps[lp] {
		if id == polledId {
			es.timestamps[lp] = append(es.timestamps[lp][:i], es.timestamps[lp][i+1:]...)
		}
	}
	es.timestamps[height] = append(es.timestamps[height], polledId)
	es.lastPoll[polledId] = height
}

func (es *EthSubscriptions) AddLogPoll(filter eth.EthFilter, height uint64) (string, error) {
	return es.Add(&EthLogPoll{
		filter:        filter,
		lastBlockRead: uint64(0),
		blockStore:    es.blockStore,
		evmAuxStore:   es.evmAuxStore,
	}, height), nil
}

func (es *EthSubscriptions) LegacyAddLogPoll(filter string, height uint64) (string, error) {
	newPoll, err := NewEthLogPoll(filter, es.evmAuxStore, es.blockStore)
	if err != nil {
		return "", err
	}
	return es.Add(newPoll, height), nil
}

func (es *EthSubscriptions) AddBlockPoll(height uint64) string {
	return es.Add(NewEthBlockPoll(height, es.evmAuxStore, es.blockStore), height)
}

func (es *EthSubscriptions) AddTxPoll(height uint64) string {
	return es.Add(NewEthTxPoll(height, es.evmAuxStore, es.blockStore), height)
}

func (es *EthSubscriptions) AllLogs(
	s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	if poll, ok := es.polls[id]; !ok {
		return nil, fmt.Errorf("subscription not found")
	} else {
		return poll.AllLogs(s, id, readReceipts)
	}
}

func (es *EthSubscriptions) Poll(
	s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	poll, ok := es.polls[id]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.Poll(s, id, readReceipts)
	es.polls[id] = newPoll

	es.resetTimestamp(id, uint64(s.Block().Height))
	return result, err

}

func (es *EthSubscriptions) LegacyPoll(
	s state.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	poll, ok := es.polls[id]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	newPoll, result, err := poll.LegacyPoll(s, id, readReceipts)
	es.polls[id] = newPoll

	es.resetTimestamp(id, uint64(s.Block().Height))
	return result, err

}

func (es *EthSubscriptions) Remove(id string) {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	delete(es.polls, id)
	delete(es.lastPoll, id)
}
