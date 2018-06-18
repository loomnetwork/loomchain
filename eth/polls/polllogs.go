// +build evm

package polls

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
)

type EthLogPoll struct {
	filter        query.EthFilter
	lastBlockRead uint64
}

func NewEthLogPoll(filter string) (*EthLogPoll, error) {
	ethFilter, err := query.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return nil, err
	}
	p := &EthLogPoll{
		filter:        ethFilter,
		lastBlockRead: uint64(0),
	}
	return p, nil
}

func (p EthLogPoll) Poll(state loomchain.ReadOnlyState, id string) (EthPoll, []byte, error) {
	start, err := query.BlockNumber(p.filter.FromBlock, uint64(state.Block().Height))
	if err != nil {
		return p, nil, err
	}
	end, err := query.BlockNumber(p.filter.ToBlock, uint64(state.Block().Height))
	if err != nil {
		return p, nil, err
	}

	if start <= p.lastBlockRead {
		start = p.lastBlockRead + 1
		if start > end {
			return p, nil, fmt.Errorf("filter start after filter end")
		}
	}

	eventLogs, err := query.GetBlockLogRange(start, end, p.filter.EthBlockFilter, state)
	if err != nil {
		return p, nil, err
	}
	newLogPoll := EthLogPoll{
		filter:        p.filter,
		lastBlockRead: end,
	}
	result, err := proto.Marshal(&types.EthFilterLogList{eventLogs})
	return newLogPoll, result, err
}
