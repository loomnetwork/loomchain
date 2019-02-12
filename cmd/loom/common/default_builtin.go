// +build !gamechain

package common

import (
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var builtinContracts []goloomplugin.Contract

func init() {
	builtinContracts = []goloomplugin.Contract{
		coin.Contract,
	}
}
