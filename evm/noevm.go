// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/state"
	lvm "github.com/loomnetwork/loomchain/vm"
)

var (
	LogEthDbBatch = true
)

// EVMEnabled indicates whether or not EVM integration is available
const EVMEnabled = false

func NewLoomVm(
	loomState state.State,
	eventHandler loomchain.EventHandler,
	receiptHandler loomchain.WriteReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
	debug bool,
) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
