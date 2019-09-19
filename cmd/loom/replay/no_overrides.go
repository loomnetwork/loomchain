// +build !basechain

package replay

import (
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
)

func ContractOverrides() plugin.ContractOverrideMap {
	return nil
}

func OverrideConfig(cfg *config.Config, blockHeight int64) *config.Config {
	return cfg
}
