package vm

import (
	"testing"
	"context"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	abci "github.com/tendermint/abci/types"
)

func TestProcessDeployTx(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:  []byte("myCaller"),
	}

	evm := NewEvm()
	//testEvents(t, evm)
	testCryptoZombies(t, evm, caller)
	testLoomTokens(t, evm, caller)

	loomstore := loom.NewStoreState(context.Background(), store.NewMemStore(), abci.Header{})
	loomevm := NewLoomEvm(loomstore)
	testCryptoZombies(t, loomevm, caller)
	testLoomTokens(t, loomevm, caller)
}

