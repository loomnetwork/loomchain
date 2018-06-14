// +build evm

package subs

import (
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
)

type EthSubscriptions struct {
	subs map[string]EthSubInfo
}

type EthSubInfo struct {
	filter        query.EthFilter
	lastBlockRead uint64
}

func NewEthSubscriptions() *EthSubscriptions {
	p := &EthSubscriptions{
		subs: make(map[string]EthSubInfo),
	}
	return p
}

func (s EthSubscriptions) Add(filter string) (string, error) {
	id := s.getId()
	ethFilter, err := query.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return "", err
	}
	s.subs[id] = EthSubInfo{
		filter:        ethFilter,
		lastBlockRead: uint64(0),
	}
	return id, nil
}

func (s EthSubscriptions) Poll(state loomchain.ReadOnlyState, id string) ([]byte, error) {
	if subInfo, ok := s.subs[id]; !ok {
		return nil, fmt.Errorf("subscripton not found")
	} else {
		filter := subInfo.filter
		start, err := query.BlockNumber(filter.FromBlock, uint64(state.Block().Height))
		if err != nil {
			return nil, err
		}
		end, err := query.BlockNumber(filter.ToBlock, uint64(state.Block().Height))
		if err != nil {
			return nil, err
		}

		if start <= subInfo.lastBlockRead {
			start = subInfo.lastBlockRead + 1
			if start > end {
				return nil, fmt.Errorf("filter start after filter end")
			}
		}

		eventLogs, err := query.GetBlockLogRange(start, end, subInfo.filter.EthBlockFilter, state)
		if err != nil {
			return nil, err
		}
		s.subs[id] = EthSubInfo{
			filter:        s.subs[id].filter,
			lastBlockRead: end,
		}
		return proto.Marshal(&types.EthFilterLogList{eventLogs})
	}

}

func (s EthSubscriptions) Remove(id string) {
	delete(s.subs, id)
}

func (s EthSubscriptions) getId() string {
	return string(rpc.NewID())
}
