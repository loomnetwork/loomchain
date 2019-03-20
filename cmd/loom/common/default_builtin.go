// +build !gamechain

package common

import (
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

func init() {
	builtinContracts = append(builtinContracts, coin.Contract)
}
