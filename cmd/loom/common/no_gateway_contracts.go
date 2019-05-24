// +build !gateway

package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/config"
)

func loadGatewayContracts(cfg *config.Config) []goloomplugin.Contract {
	return []goloomplugin.Contract{}
}
