// +build !evm

package gateway

import (
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

type GatewayType int

const (
	EthereumGateway GatewayType = 0 // default type
	LoomCoinGateway GatewayType = 1
	TronGateway     GatewayType = 2
)

type (
	InitRequest = tgtypes.TransferGatewayInitRequest
)

type Gateway struct {
	Type GatewayType
}

type UnsafeGateway struct {
	Gateway
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	switch gw.Type {
	case EthereumGateway:
		return plugin.Meta{
			Name:    "gateway",
			Version: "0.1.0",
		}, nil
	case LoomCoinGateway:
		return plugin.Meta{
			Name:    "loomcoin-gateway",
			Version: "0.1.0",
		}, nil
	case TronGateway:
		return plugin.Meta{
			Name:    "tron-gateway",
			Version: "0.1.0",
		}, nil
	}
	return plugin.Meta{}, errors.Errorf("invalid Gateway Type: %v", gw.Type)
}

func (gw *Gateway) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
var UnsafeContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{}})

var LoomCoinContract plugin.Contract = contract.MakePluginContract(&Gateway{})
var UnsafeLoomCoinContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{}})

var TronContract plugin.Contract = contract.MakePluginContract(&Gateway{})
var UnsafeTronContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{}})
