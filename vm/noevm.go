// +build !evm

package vm

import (
	loom "github.com/loomnetwork/loomchain"
)

var LoomEvmFactory func(state loom.State) VM
var EvmFactory func(state loom.State) VM
