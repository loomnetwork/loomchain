package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
	"github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash"
	"github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/cmd/loom/replay"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
)

func NewDefaultContractsLoader(cfg *config.Config) plugin.Loader {
	contracts := []goloomplugin.Contract{}
	//For a quick way for other chains to just build new contracts into loom, like gamechain
	contracts = append(contracts, builtinContracts...)

	if cfg.DPOSVersion == 3 {
		contracts = append(contracts, dposv3.Contract)
	} else if cfg.DPOSVersion == 2 {
		//We need to load both dposv3 and dposv2 for migration
		contracts = append(contracts, dposv2.Contract, dposv3.Contract)
	}

	if cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts, plasma_cash.Contract)
	}
	if cfg.Karma.Enabled || cfg.Karma.ContractEnabled {
		contracts = append(contracts, karma.Contract)
	}
	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts, ethcoin.Contract)
	}
	if cfg.ChainConfig.ContractEnabled {
		contracts = append(contracts, chainconfig.Contract)
	}
	if cfg.DeployerWhitelist.ContractEnabled {
		contracts = append(contracts, deployer_whitelist.Contract)
	}
	if cfg.UserDeployerWhitelist.ContractEnabled {
		contracts = append(contracts, user_deployer_whitelist.Contract)
	}

	if cfg.AddressMapperContractEnabled() {
		contracts = append(contracts, address_mapper.Contract)
	}

	contracts = append(contracts, loadGatewayContracts(cfg)...)

	loader := plugin.NewStaticLoader(contracts...)
	loader.SetContractOverrides(replay.ContractOverrides())
	return loader
}
