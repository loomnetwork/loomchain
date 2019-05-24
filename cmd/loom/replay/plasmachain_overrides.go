// +build plasmachain

package replay

import (
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
	gateway_v1 "github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway/v1"
)

func ContractOverrides() plugin.ContractOverrideMap {
	return plugin.ContractOverrideMap{
		"gateway:0.1.0": []*plugin.ContractOverride{
			&plugin.ContractOverride{
				Contract:    gateway_v1.Contract,
				BlockHeight: 1,
			},
			&plugin.ContractOverride{
				Contract:    gateway.Contract,
				BlockHeight: 197576,
			},
		},
	}
}

func OverrideConfig(cfg *config.Config, blockHeight int64) *config.Config {
	if blockHeight < 197576 { // build 424
		clone := cfg.Clone()
		clone.ReceiptsVersion = handler.ReceiptHandlerLegacyV1
		return clone
	} else if (blockHeight == 197576) || (blockHeight == 197577) { // build 430
		clone := cfg.Clone()
		clone.DeployEnabled = false
		clone.CallEnabled = false
		clone.ReceiptsVersion = handler.ReceiptHandlerLegacyV1
		return clone
	} else if blockHeight < 356720 { // build 430
		clone := cfg.Clone()
		clone.ReceiptsVersion = handler.ReceiptHandlerLegacyV1
		return clone
	} else if blockHeight < 548320 { // build 495
		clone := cfg.Clone()
		clone.ReceiptsVersion = handler.ReceiptHandlerLegacyV2
		return clone
	}
	return cfg
}
