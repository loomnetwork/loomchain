package user_deployer_whitelist

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type UserDeployerWhitelist struct {
}

func (uw *UserDeployerWhitelist) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "user-deployer-whitelist",
		Version: "1.0.0",
	}, nil
}

func RecordContractDeployment(ctx contract.Context, deployerAddress loom.Address, contractAddr loom.Address) error {

}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
