// +build !evm

package evm

import (
	"github.com/loomnetwork/loomchain"
	lvm "github.com/loomnetwork/loomchain/vm"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	LogEthDbBatch = true
)

// EVMEnabled indicates whether or not EVM integration is available
const EVMEnabled = false

func NewLoomVm(
	loomState loomchain.State,
	evmDB dbm.DB,
	eventHandler loomchain.EventHandler,
	receiptHandler loomchain.WriteReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
	debug bool,
) lvm.VM {
	return nil
}

func AddLoomPrecompiles() {}
