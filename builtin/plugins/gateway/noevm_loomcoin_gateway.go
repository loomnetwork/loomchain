// +build !evm

package gateway

import (
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type LoomcoinGateway struct {
}

func (gw *LoomcoinGateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "loomcoin-gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *LoomcoinGateway) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

var LoomCoinContract plugin.Contract = contract.MakePluginContract(&LoomcoinGateway{})
