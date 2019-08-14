// +build gateway

package main

import (
	"fmt"
	"time"

	glAuth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/fnConsensus"
	"github.com/loomnetwork/loomchain/log"
	tgateway "github.com/loomnetwork/transfer-gateway/gateway"
)

type ServiceName string

const (
	GatewayName        ServiceName = "gateway"
	LoomGatewayName    ServiceName = "loomcoin-gateway"
	BinanceGatewayName ServiceName = "binance-gateway"
	TronGatewayName    ServiceName = "tron-gateway"
)

func startGatewayReactors(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *config.Config,
	nodeSigner glAuth.Signer,
) error {
	if err := startGatewayFn(chainID, fnRegistry, cfg.TransferGateway, nodeSigner); err != nil {
		return err
	}

	if err := startLoomCoinGatewayFn(chainID, fnRegistry, cfg.LoomCoinTransferGateway, nodeSigner); err != nil {
		return err
	}

	if err := startTronGatewayFn(chainID, fnRegistry, cfg.TronTransferGateway, nodeSigner); err != nil {
		return err
	}

	return nil
}

// Checks if the query server at the given DAppChainReadURI is responding.
func checkQueryService(name ServiceName, chainID string, DAppChainReadURI string, DAppChainWriteURI string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		rpcClient := client.NewDAppChainRPCClient(chainID, DAppChainWriteURI, DAppChainReadURI)
		gatewayAddr, err := rpcClient.Resolve(fmt.Sprintf("%s", name))
		if err != nil {
			log.Error("Error while resolving service", DAppChainReadURI, err)
		} else {
			log.Info(fmt.Sprintf("Resolved %s : %#v", name, gatewayAddr))
			return
		}
	}
}

func startGatewayFn(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *tgateway.TransferGatewayConfig,
	nodeSigner glAuth.Signer,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}

	checkQueryService(GatewayName, chainID, cfg.DAppChainReadURI, cfg.DAppChainWriteURI)
	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(false, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("batch_sign_withdrawal", batchSignWithdrawalFn)
}

func startLoomCoinGatewayFn(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *tgateway.TransferGatewayConfig,
	nodeSigner glAuth.Signer,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}

	checkQueryService(LoomGatewayName, chainID, cfg.DAppChainReadURI, cfg.DAppChainWriteURI)
	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("loomcoin:batch_sign_withdrawal", batchSignWithdrawalFn)
}

func startTronGatewayFn(chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *tgateway.TransferGatewayConfig,
	nodeSigner glAuth.Signer,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}

	checkQueryService(TronGatewayName, chainID, cfg.DAppChainReadURI, cfg.DAppChainWriteURI)
	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("tron:batch_sign_withdrawal", batchSignWithdrawalFn)
}
