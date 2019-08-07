// +build gateway

package main

import (
	"fmt"
	"net/http"
	"time"

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
	if err := startGatewayFn(chainID, fnRegistry,
		cfg.TransferGateway, nodeSigner, cfg.QueryServicePollTimeout); err != nil {
		return err
	}

	if err := startLoomCoinGatewayFn(chainID, fnRegistry,
		cfg.LoomCoinTransferGateway, nodeSigner, cfg.QueryServicePollTimeout); err != nil {
		return err
	}

	if err := startTronGatewayFn(chainID, fnRegistry,
		cfg.TronTransferGateway, nodeSigner, cfg.QueryServicePollTimeout); err != nil {
		return err
	}

	return nil
}

// checkQueryService is a helper function to check if corresponding QueryService has started
// timeout is configurable with variable QueryServicePollTimeout
func checkQueryService(url string, timeout int) error {
	timeoutTick := time.After(time.Duration(timeout) * time.Millisecond)
	tick := time.Tick(1000 * time.Millisecond)
	for {
		select {
		case <-tick:
			var netClient = &http.Client{
				Timeout: time.Second * 1,
			}
			resp, _ := netClient.Head(url)
			if resp != nil && resp.StatusCode == 200 {
				return nil
			}
		case <-timeoutTick:
			return fmt.Errorf("Unable to connect to queryserver at %s", url)
		}
	}
}

func startGatewayFn(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *tgateway.TransferGatewayConfig,
	nodeSigner glAuth.Signer,
	queryServicePollTimeout int,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}
	if err := checkQueryService(cfg.DAppChainReadURI, queryServicePollTimeout); err != nil {
		return err
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
	queryServicePollTimeout int,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}
	if err := checkQueryService(cfg.DAppChainReadURI, queryServicePollTimeout); err != nil {
		return err
	}
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
	queryServicePollTimeout int,
) error {
	if !cfg.BatchSignFnConfig.Enabled {
		return nil
	}

	if fnRegistry == nil {
		return fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}
	if err := checkQueryService(cfg.DAppChainReadURI, queryServicePollTimeout); err != nil {
		return err
	}
	batchSignWithdrawalFn, err := tgateway.CreateBatchSignWithdrawalFn(true, chainID, cfg, nodeSigner)
	if err != nil {
		return err
	}

	return fnRegistry.Set("tron:batch_sign_withdrawal", batchSignWithdrawalFn)
}
