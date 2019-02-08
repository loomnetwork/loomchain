// +build gamechain

package common

import (
	"github.com/loomnetwork/gamechain/battleground"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
)

var builtinContracts []goloomplugin.Contract

func init() {
	builtinContracts = []goloomplugin.Contract{
		battleground.Contract,
	}
}
