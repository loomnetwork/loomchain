package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash"
	"github.com/loomnetwork/loomchain/cmd/loom/replay"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
)

func NewDefaultContractsLoader(cfg *config.Config) plugin.Loader {
	contracts := []goloomplugin.Contract{
		coin.Contract,
	}
	if cfg.DPOSVersion == 2 {
		contracts = append(contracts, dposv2.Contract)
	} else {
		contracts = append(contracts, dpos.Contract)
	}
	if cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts, plasma_cash.Contract)
	}
	if cfg.KarmaEnabled {
		contracts = append(contracts, karma.Contract)
	}
	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts, ethcoin.Contract)
	}
	if cfg.TransferGateway.ContractEnabled || cfg.LoomCoinTransferGateway.ContractEnabled || cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts, address_mapper.Contract)
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts, gateway.Contract)
	}

	if cfg.LoomCoinTransferGateway.ContractEnabled {
		contracts = append(contracts, gateway.LoomCoinContract)
	}
	
	loader := plugin.NewStaticLoader(contracts...)
	loader.SetContractOverrides(replay.ContractOverrides())
	return loader
}
