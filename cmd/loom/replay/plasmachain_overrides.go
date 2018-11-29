// +build plasmachain

package replay

import (
	"github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	gateway_v1 "github.com/loomnetwork/loomchain/builtin/plugins/gateway/v1"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
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
	if (blockHeight == 197576) || (blockHeight == 197577) {
		clone := cfg.Clone()
		clone.DeployEnabled = false
		clone.CallEnabled = false
	}
	return cfg
}
