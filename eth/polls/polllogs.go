// +build evm

package polls

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

type EthLogPoll struct {
	filter        eth.EthFilter
	lastBlockRead uint64
	evmAuxStore   *evmaux.EvmAuxStore
	blockStore    store.BlockStore
}

func NewEthLogPoll(filter string, evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) (*EthLogPoll, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(filter))
	if err != nil {
		return nil, err
	}
	p := &EthLogPoll{
		filter:        ethFilter,
		lastBlockRead: uint64(0),
		evmAuxStore:   evmAuxStore,
		blockStore:    blockStore,
	}
	return p, nil
}

func (p *EthLogPoll) Poll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (EthPoll, interface{}, error) {
	start, err := eth.DecBlockHeight(state.Block().Height, p.filter.FromBlock)
	if err != nil {
		return p, nil, err
	}
	end, err := eth.DecBlockHeight(state.Block().Height, p.filter.ToBlock)
	if err != nil {
		return p, nil, err
	}

	if start > end {
		return p, nil, errors.New("Filter FromBlock is greater than ToBlock")
	}

	if start <= p.lastBlockRead {
		start = p.lastBlockRead + 1
		if start > end {
			return p, nil, nil
		}
	}

	eventLogs, err := query.GetBlockLogRange(
		p.blockStore, state, start, end, p.filter.EthBlockFilter, readReceipts, p.evmAuxStore,
	)
	if err != nil {
		return p, nil, err
	}
	newLogPoll := &EthLogPoll{
		filter:        p.filter,
		lastBlockRead: end,
	}
	return newLogPoll, eth.EncLogs(eventLogs), nil
}

func (p *EthLogPoll) AllLogs(state loomchain.ReadOnlyState,
	id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error) {
	start, err := eth.DecBlockHeight(state.Block().Height, p.filter.FromBlock)
	if err != nil {
		return nil, err
	}
	end, err := eth.DecBlockHeight(state.Block().Height, p.filter.ToBlock)
	if err != nil {
		return nil, err
	}
	if start > end {
		return nil, errors.New("Filter FromBlock is greater than ToBlock")
	}

	eventLogs, err := query.GetBlockLogRange(
		p.blockStore, state, start, end, p.filter.EthBlockFilter, readReceipts, p.evmAuxStore,
	)
	if err != nil {
		return nil, err
	}
	return eth.EncLogs(eventLogs), nil
}

func (p *EthLogPoll) LegacyPoll(state loomchain.ReadOnlyState,
	id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	start, err := eth.DecBlockHeight(state.Block().Height, p.filter.FromBlock)
	if err != nil {
		return p, nil, err
	}

	end, err := eth.DecBlockHeight(state.Block().Height, p.filter.ToBlock)
	if err != nil {
		return p, nil, err
	}

	if start <= p.lastBlockRead {
		start = p.lastBlockRead + 1
		if start > end {
			return p, nil, fmt.Errorf("filter start after filter end")
		}
	}
	eventLogs, err := query.GetBlockLogRange(
		p.blockStore, state, start, end, p.filter.EthBlockFilter, readReceipts, p.evmAuxStore,
	)
	if err != nil {
		return p, nil, err
	}
	newLogPoll := &EthLogPoll{
		filter:        p.filter,
		lastBlockRead: end,
		evmAuxStore:   p.evmAuxStore,
		blockStore:    p.blockStore,
	}

	blocksMsg := types.EthFilterEnvelope_EthFilterLogList{
		EthFilterLogList: &types.EthFilterLogList{EthBlockLogs: eventLogs},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return newLogPoll, r, err
}
