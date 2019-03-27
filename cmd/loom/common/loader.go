package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash"
	"github.com/loomnetwork/loomchain/cmd/loom/replay"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
)

func NewDefaultContractsLoader(cfg *config.Config) plugin.Loader {
	contracts := []goloomplugin.Contract{}
	//For a quick way for other chains to just build new contracts into loom, like gamechain
	for _, cnt := range builtinContracts {
		contracts = append(contracts, cnt)
	}
	if cfg.DPOSVersion == 3 {
		contracts = append(contracts, dposv3.Contract)
	} else if cfg.DPOSVersion == 2 {
		contracts = append(contracts, dposv2.Contract)
	}
	if cfg.DPOSVersion == 1 || cfg.BootLegacyDPoS == true {
		//Plasmachain or old legacy chain need dposv1 to be able to bootstrap the chain.
		contracts = append(contracts, dpos.Contract)
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

	if cfg.AddressMapperContractEnabled() {
		contracts = append(contracts, address_mapper.Contract)
	}

	if cfg.TransferGateway.ContractEnabled {
		if cfg.TransferGateway.Unsafe {
			contracts = append(contracts, gateway.UnsafeContract)
		} else {
			contracts = append(contracts, gateway.Contract)
		}
	}

	if cfg.LoomCoinTransferGateway.ContractEnabled {
		if cfg.LoomCoinTransferGateway.Unsafe {
			contracts = append(contracts, gateway.UnsafeLoomCoinContract)
		} else {
			contracts = append(contracts, gateway.LoomCoinContract)
		}
	}

	loader := plugin.NewStaticLoader(contracts...)
	loader.SetContractOverrides(replay.ContractOverrides())
	return loader
}
