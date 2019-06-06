// +build gateway

package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func loadGatewayContracts(cfg *config.Config) []goloomplugin.Contract {
	contracts := []goloomplugin.Contract{}

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

	if cfg.TronTransferGateway.ContractEnabled {
		if cfg.TronTransferGateway.Unsafe {
			contracts = append(contracts, gateway.UnsafeTronContract)
		} else {
			contracts = append(contracts, gateway.TronContract)
		}
	}

	return contracts
}
