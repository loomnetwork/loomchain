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
	_ state.State,
	_ loomchain.EventHandler,
	_ loomchain.WriteReceiptHandler,
	_ AccountBalanceManagerFactoryFunc,
	_ bool,
	_ *eth.TraceConfig,
) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
