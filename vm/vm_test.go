package vm

import (
	"testing"
	"github.com/loomnetwork/loom"
)

func TestProcessDeployTx(t *testing.T) {
	evm := NewEvm()
	caller := loom.Address{
		ChainID: "myChainID",
		Local:  []byte("myCaller"),
	}

	//testEvents(t, evm)
	testCryptoZombies(t, evm, caller)
	testLoomTokens(t, evm, caller)
}

