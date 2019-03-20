// +build !gamechain !plasmachain

package common

import (
	"github.com/loomnetwork/weave-blueprint/src/blueprint"
)

func init() {
	builtinContracts = append(builtinContracts, blueprint.Contract)
}
