// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	lvm "github.com/loomnetwork/loomchain/vm"
)

var (
	LogEthDbBatch          = true
	GasUsageTrackerEnabled = false
)

// EVMEnabled indicates whether or not EVM integration is available
const (
	EVMEnabled = false
)

func NewLoomVm(
	loomState loomchain.State,
	eventHandler loomchain.EventHandler,
	receiptHandler loomchain.WriteReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
	debug bool,
) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}

func GetGasUsage(addr string) uint64 { return 0 }
