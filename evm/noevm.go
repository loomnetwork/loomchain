// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	lvm "github.com/loomnetwork/loomchain/vm"
)

var (
	LogEthDbBatch = true
)

// EVMEnabled indicates whether or not EVM integration is available
const EVMEnabled = false

func NewLoomVm(
		loomState       loomchain.State,
		receiptCache    *loomchain.WriteReceiptCache,
		createABM       AccountBalanceManagerFactoryFunc,
	) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
