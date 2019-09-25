// +build gateway

package main

import (
	"fmt"
	"time"

	glAuth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	tgateway "github.com/loomnetwork/transfer-gateway/gateway"
	tconfig "github.com/loomnetwork/transfer-gateway/gateway/config"

	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/fnConsensus"
	"github.com/loomnetwork/loomchain/log"
)

const (
	GatewayName        = "gateway"
	LoomGatewayName    = "loomcoin-gateway"
	BinanceGatewayName = "binance-gateway"
	TronGatewayName    = "tron-gateway"
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
func checkQueryService(name string, chainID string, DAppChainReadURI string, DAppChainWriteURI string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		rpcClient := client.NewDAppChainRPCClient(chainID, DAppChainWriteURI, DAppChainReadURI)
		gatewayAddr, err := rpcClient.Resolve(name)
		if err != nil {
			log.Error("Failed to resolve gateway contract", "gateway", name, "err", err)
		} else {
			log.Debug("Resolved gateway contract", "gateway", name, "addr", gatewayAddr)
			return
		}
	}
}

func startGatewayFn(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *tconfig.TransferGatewayConfig,
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
	cfg *tconfig.TransferGatewayConfig,
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
	cfg *tconfig.TransferGatewayConfig,
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
