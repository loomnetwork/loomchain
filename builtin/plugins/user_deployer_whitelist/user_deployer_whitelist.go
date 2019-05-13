package user_deployer_whitelist

import "github.com/loomnetwork/go-loom/plugin"
import contract "github.com/loomnetwork/go-loom/plugin/contractpb"

type UserDeployerWhitelist struct {
}

func (uw *UserDeployerWhitelist) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "userdeployerwhitelist",
		Version: "1.0.0",
	}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
