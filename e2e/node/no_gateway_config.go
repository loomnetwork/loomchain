// +build !gateway

package node

import (
	"github.com/loomnetwork/loomchain/config"
)

func configureGateways(cfg *config.Config, proxyAppPort int) {
	// noop
}
