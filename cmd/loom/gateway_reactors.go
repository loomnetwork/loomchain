// +build gateway

package main

import (
	"fmt"

	glAuth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/fnConsensus"
	tgateway "github.com/loomnetwork/transfer-gateway/gateway"
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

	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("loomcoin:batch_sign_withdrawal", batchSignWithdrawalFn)
}

func startTronGatewayFn(chainID string, fnRegistry fnConsensus.FnRegistry, cfg *tgateway.TransferGatewayConfig, nodeSigner glAuth.Signer) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}

	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("tron:batch_sign_withdrawal", batchSignWithdrawalFn)
}
