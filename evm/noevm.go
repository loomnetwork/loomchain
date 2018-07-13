// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	lvm "github.com/loomnetwork/loomchain/vm"
)

var (
	LogEthDbBatch = true
)

var LoomVmFactory func(state loomchain.State) lvm.VM

func NewLoomVm(state loomchain.State, eventHandler loomchain.EventHandler) lvm.VM { return nil }

func AddLoomPrecompiles() {}
