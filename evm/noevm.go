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
	_ loomchain.State,
	_ loomchain.WriteReceiptHandler,
	_ AccountBalanceManagerFactoryFunc,
	_ bool,
	_ interface{},
) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
