// +build evm

package polls

import (
	"fmt"

	"github.com/loomnetwork/loomchain/receipts"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/eth/utils"
)

type EthLogPoll struct {
	filter        utils.EthFilter
	lastBlockRead uint64
}

func NewEthLogPoll(filter string) (*EthLogPoll, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return nil, err
	}
	p := &EthLogPoll{
		filter:        ethFilter,
		lastBlockRead: uint64(0),
	}
	return p, nil
}

func (p EthLogPoll) Poll(state loomchain.ReadOnlyState, id string, readReceipts receipts.ReadReceiptHandler) (EthPoll, []byte, error) {
	start, err := utils.BlockNumber(p.filter.FromBlock, uint64(state.Block().Height))
	if err != nil {
		return p, nil, err
	}
	end, err := utils.BlockNumber(p.filter.ToBlock, uint64(state.Block().Height))
	if err != nil {
		return p, nil, err
	}

	if start <= p.lastBlockRead {
		start = p.lastBlockRead + 1
		if start > end {
			return p, nil, fmt.Errorf("filter start after filter end")
		}
	}

	eventLogs, err := query.GetBlockLogRange(start, end, p.filter.EthBlockFilter, readReceipts)
	if err != nil {
		return p, nil, err
	}
	newLogPoll := EthLogPoll{
		filter:        p.filter,
		lastBlockRead: end,
	}

	blocksMsg := types.EthFilterEnvelope_EthFilterLogList{
		&types.EthFilterLogList{EthBlockLogs: eventLogs},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return newLogPoll, r, err
}
