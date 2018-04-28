// +build !evm

package vm

import (
	"github.com/loomnetwork/loomchain"
)

var LoomEvmFactory func(state loomchain.State) VM
var EvmFactory func(state loomchain.State) VM
