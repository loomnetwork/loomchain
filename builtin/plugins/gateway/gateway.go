package gateway

import (
	"errors"
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	stateKey = []byte("state")

	errERC20TransferFailed = errors.New("failed to call ERC20 Transfer method")

	// Permissions
	changeMappingsPerm = []byte("change-mappings")
	changeOraclesPerm  = []byte("change-oracles")
	submitEventsPerm   = []byte("submit-events")

	// Roles
	ownerRole  = "owner"
	oracleRole = "oracle"
)

var (
	// ErrNotAuthorized indicates that a contract method failed because the caller didn't have the permission to execute that method.
	ErrNotAuthorized = errors.New("not authorized")
)

func tokenKey(tokenContractAddr loom.Address) []byte {
	return util.PrefixKey([]byte("token"), tokenContractAddr.Bytes())
}

// TODO: list of oracles should be editable, the genesis should contain the initial set

// Gateway structure for the Plugin
type Gateway struct {
}

// Meta information about the plugin
func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
}

// Init initializes the plugin
func (gw *Gateway) Init(ctx contract.Context, req *GatewayInitRequest) error {
	ctx.GrantPermission(changeOraclesPerm, []string{ownerRole})
	ctx.GrantPermission(changeMappingsPerm, []string{ownerRole})

	// TODO: Find a good way to set the oracle address
	for _, oracleAddr := range req.Oracles {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(oracleAddr), submitEventsPerm, oracleRole)
	}

	for _, tokenMapping := range req.Tokens {
		ctx.Set(tokenKey(loom.UnmarshalAddressPB(tokenMapping.FromToken)), tokenMapping)
	}

	state := &GatewayState{
		LastEthBlock: 0,
	}
	return ctx.Set(stateKey, state)
}

// ProcessEventBatch processes ERC20/ERC721 deposits
func (gw *Gateway) ProcessEventBatch(ctx contract.Context, req *ProcessEventBatchRequest) error {
	// TODO: Oracle permission commented for while
	// if ok, _ := ctx.HasPermission(submitEventsPerm, []string{oracleRole}); !ok {
	// 	return ErrNotAuthorized
	// }

	state, err := gw.loadState(ctx)
	if err != nil {
		return err
	}

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

		// TODO: figure out if it's a good idea to process the rest of the deposits if one fails
		if err = gw.transferTokenDeposit(ctx, ftd); err != nil {
			ctx.Logger().Error(err.Error())
			continue
		}

		if ftd.EthBlock > lastEthBlock {
			blockCount++
			lastEthBlock = ftd.EthBlock
		}
	}

	// TODO: process NFT deposits

	// If there are no new events in this batch return an error so that the batch tx isn't
	// propagated to the other nodes.
	if blockCount == 0 {
		return fmt.Errorf("no new events found in the batch")
	}

	state.LastEthBlock = lastEthBlock

	return ctx.Set(stateKey, state)
}

// GetState the current state of the gateway
func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	state, err := gw.loadState(ctx)
	if err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: state}, nil
}

// AddTokenMapping adds a new mappings of external and internal user token
func (gw *Gateway) AddTokenMapping(ctx contract.Context, req *GatewayTokenMapping) error {
	// TODO: Set the permission system correctly
	// if ok, _ := ctx.HasPermission(changeMappingsPerm, []string{ownerRole}); !ok {
	// 	return ErrNotAuthorized
	// }

	ctx.Set(tokenKey(loom.UnmarshalAddressPB(req.FromToken)), req.ToToken)
	return nil
}

// RemoveTokenMapping remove mapping of external and internal user token
func (gw *Gateway) RemoveTokenMapping(ctx contract.Context, req *GatewayTokenMapping) error {
	// TODO: Set the permission system correctly
	// if ok, _ := ctx.HasPermission(changeMappingsPerm, []string{ownerRole}); !ok {
	// 	return ErrNotAuthorized
	// }

	ctx.Delete(tokenKey(loom.UnmarshalAddressPB(req.FromToken)))
	return nil
}

func (gw *Gateway) transferTokenDeposit(ctx contract.Context, ftd *TokenDeposit) error {
	fromTokenAddr := loom.UnmarshalAddressPB(ftd.From)
	var gatewayTokenMapping GatewayTokenMapping
	err := ctx.Get(tokenKey(fromTokenAddr), &gatewayTokenMapping)
	if err != nil {
		return fmt.Errorf("failed to map token %v to DAppChain coin", fromTokenAddr.String())
	}

	addr, err := ctx.Resolve(gatewayTokenMapping.CoinName)
	if err != nil {
		return fmt.Errorf("failed to get address for %s", gatewayTokenMapping.CoinName)
	}

	err = contract.CallMethod(ctx, addr, "Mint", &coin.MintRequest{
		To:     ftd.To,
		Amount: ftd.Amount,
	}, nil)

	if err != nil {
		ctx.Logger().Error(errERC20TransferFailed.Error(), "err", err)
		return errERC20TransferFailed
	}
	return nil
}

func (gw *Gateway) loadState(ctx contract.StaticContext) (*GatewayState, error) {
	var state GatewayState
	err := ctx.Get(stateKey, &state)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}
	return &state, nil
}

// Contract plugin
var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
