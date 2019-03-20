// +build gamechain

package common

import (
	"github.com/loomnetwork/gamechain/battleground"
)

func init() {
	builtinContracts = append(builtinContracts, battleground.Contract)
}
