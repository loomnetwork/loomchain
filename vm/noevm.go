// +build !evm

package vm

import (
	"github.com/loomnetwork/loomchain"
)

var (
	ReceiptPrefix = []byte("receipt")
)

var LoomEvmFactory func(state loomchain.State) VM
var EvmFactory func(state loomchain.State) VM
var LoomVmFactory func(state loomchain.State) VM

func NewLoomVm(state loomchain.State, eventHandler loomchain.EventHandler) VM { return nil }
