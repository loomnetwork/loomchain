package gateway

import (
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
)

var (
	stateKey = []byte("state")
)

func tokenKey(tokenContractAddr loom.Address) []byte {
	return util.PrefixKey([]byte("token"), tokenContractAddr.Bytes())
}

type Gateway struct {
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *Gateway) Init(ctx contract.Context, req *GatewayInitRequest) error {
	for _, tokenMapping := range req.Tokens {
		ctx.Set(tokenKey(loom.UnmarshalAddressPB(tokenMapping.FromToken)), tokenMapping.ToToken)
	}
	state := &GatewayState{
		LastEthBlock: 0,
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

		fromTokenAddr := loom.UnmarshalAddressPB(ftd.Token)
		var toTokenAddrPB types.Address
		err := ctx.Get(tokenKey(fromTokenAddr), &toTokenAddrPB)
		if err != nil {
			return fmt.Errorf("failed to map token %v to DAppChain token", fromTokenAddr.String())
		}
		toTokenAddr := loom.UnmarshalAddressPB(&toTokenAddrPB)

		err = contract.CallMethod(ctx, toTokenAddr, "Transfer", &coin.TransferRequest{
			To:     ftd.To,
			Amount: ftd.Amount,
		}, nil)

		// TODO: figure out if it's a good idea to process the rest of the deposits if one fails
		if err != nil {
			ctx.Logger().Error(err.Error())
			continue
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

	return ctx.Set(stateKey, &state)
}

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	var state GatewayState
	err := ctx.Get(stateKey, &state)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}
	return &GatewayStateResponse{State: &state}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
