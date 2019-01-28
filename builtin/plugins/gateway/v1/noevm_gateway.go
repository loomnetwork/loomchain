// +build !evm

package gateway

import (
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway/v1"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type (
	InitRequest = tgtypes.TransferGatewayInitRequest
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

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
