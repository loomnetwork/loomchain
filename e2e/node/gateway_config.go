// +build gateway

package node

import (
	"fmt"

	"github.com/loomnetwork/loomchain/config"
)

func configureGateways(cfg *config.Config, proxyAppPort int) {
	cfg.LoomCoinTransferGateway.DAppChainReadURI = fmt.Sprintf("http://127.0.0.1:%d/query", proxyAppPort)
	cfg.LoomCoinTransferGateway.DAppChainWriteURI = fmt.Sprintf("http://127.0.0.1:%d/rpc", proxyAppPort)
	cfg.TransferGateway.DAppChainReadURI = fmt.Sprintf("http://127.0.0.1:%d/query", proxyAppPort)
	cfg.TransferGateway.DAppChainWriteURI = fmt.Sprintf("http://127.0.0.1:%d/rpc", proxyAppPort)
}
