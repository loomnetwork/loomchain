// +build !evm

package plasma_cash

import (
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

// Dummy file so you can build the server without the EVM

type PlasmaCash struct {
}

func (c *PlasmaCash) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "plasmacash",
		Version: "1.0.0",
	}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
