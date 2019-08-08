// +build gateway

package main

import (
	"fmt"
	"net/http"
	"time"

	glAuth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/fnConsensus"
	"github.com/loomnetwork/loomchain/log"
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

// checkQueryService is a helper function to check if corresponding QueryService has started
// timeout is configurable with variable QueryServicePollTimeout
func checkQueryService(url string) {
	ticker := time.NewTicker(5000 * time.Millisecond)
	var netClient = &http.Client{
		Timeout: time.Second * 1,
	}
	// to have the first tick immediately
	for ; true; <-ticker.C {
		resp, err := netClient.Head(url)
		if err != nil {
			log.Error("Error in connecting to queryserver:", url, err)
			continue
		}
		if resp != nil && resp.StatusCode == 200 {
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
	checkQueryService(cfg.DAppChainReadURI)
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
	checkQueryService(cfg.DAppChainReadURI)
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
	checkQueryService(cfg.DAppChainReadURI)
	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("tron:batch_sign_withdrawal", batchSignWithdrawalFn)
}
