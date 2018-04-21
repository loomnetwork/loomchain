// +build !evm

package vm

import (
	"github.com/loomnetwork/loom"
)

var LoomEvmFactory func(state loom.State) VM
var EvmFactory func(state loom.State) VM
