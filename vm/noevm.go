// +build !evm

package vm

import (
	"github.com/loomnetwork/loomchain"
)

var LoomEvmFactory func(state loom.State) VM
var EvmFactory func(state loom.State) VM
