package gateway

import (
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

var (
	stateKey = []byte("state")
)

type Gateway struct {
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *Gateway) Init(ctx contract.Context, req *GatewayInitRequest) error {
	state := &GatewayState{
		LastEthBlock: 0,
		EthBalance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
		Erc20Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
	}
	return ctx.Set(stateKey, state)
}

func (gw *Gateway) ProcessEventBatchRequest(ctx contract.Context, req *ProcessEventBatchRequest) error {
	var state GatewayState
	if err := ctx.Get(stateKey, &state); err != nil {
		return err
	}

	// TODO: transfer tokens to the corresponding coin contract on the DAppChain
	// For now just track total deposits...
	blockCount := 0           // number of blocks that were actually processed in this batch
	lastEthBlock := uint64(0) // the last block processed in this batch
	ethBal := state.EthBalance.Value
	erc20Bal := state.Erc20Balance.Value
	ethAddr := loom.RootAddress("eth")
	for _, ftd := range req.FtDeposits {
		// Events in the batch are expected to be ordered by block, so a batch should contain
		// events from block N, followed by events from block N+1, any other order is invalid.
		if ftd.EthBlock < lastEthBlock {
			return fmt.Errorf("invalid batch, block %v has already been processed", ftd.EthBlock)
		}

		// Multiple validators might submit batches with overlapping block ranges because the
		// Gateway oracles will fetch events from Ethereum at different times, with different
		// latencies, etc. Simply skip blocks that have already been processed.
		if ftd.EthBlock <= state.LastEthBlock {
			continue
		}

		tokenAddr := loom.UnmarshalAddressPB(ftd.Token)
		if ethAddr.Compare(tokenAddr) == 0 {
			ethBal.Add(&ethBal, &ftd.Amount.Value)
		} else {
			erc20Bal.Add(&erc20Bal, &ftd.Amount.Value)
		}

		if ftd.EthBlock > lastEthBlock {
			blockCount++
			lastEthBlock = ftd.EthBlock
		}
	}

	// If there are no new events in this batch return an error so that the batch tx isn't
	// propagated to the other nodes.
	if blockCount == 0 {
		return fmt.Errorf("no new events found in the batch")
	}

	state.LastEthBlock = lastEthBlock
	state.EthBalance.Value = ethBal
	state.Erc20Balance.Value = erc20Bal

	return ctx.Set(stateKey, &state)
}

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	var state GatewayState
	if err := ctx.Get(stateKey, &state); err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: &state}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
