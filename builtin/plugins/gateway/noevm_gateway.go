// +build !evm

package gateway

import (
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type (
	InitRequest              = tgtypes.GatewayInitRequest
	GatewayState             = tgtypes.GatewayState
	ProcessEventBatchRequest = tgtypes.ProcessEventBatchRequest
	GatewayStateRequest      = tgtypes.GatewayStateRequest
	GatewayStateResponse     = tgtypes.GatewayStateResponse
	NFTDeposit               = tgtypes.NFTDeposit
	TokenDeposit             = tgtypes.TokenDeposit
	TokenMapping             = tgtypes.GatewayTokenMapping
)

type Gateway struct {
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *Gateway) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

func (gw *Gateway) ProcessEventBatch(ctx contract.Context, req *ProcessEventBatchRequest) error {
	return nil
}

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	return &GatewayStateResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
