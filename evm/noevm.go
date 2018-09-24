// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	`github.com/loomnetwork/loomchain/receipts/factory`
	lvm "github.com/loomnetwork/loomchain/vm"
)

var (
	LogEthDbBatch = true
)

// EVMEnabled indicates whether or not EVM integration is available
const EVMEnabled = false

func NewLoomVm(
		loomState loomchain.State,
		eventHandler loomchain.EventHandler,
		createRecieptHandler factory.ReceiptHandlerFactoryFunc,
		createABM AccountBalanceManagerFactoryFunc,
	) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
