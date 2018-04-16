package vm

import (
	"context"
	"testing"

	abci "github.com/tendermint/abci/types"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	"github.com/stretchr/testify/require"
)

func mockState() loom.State {
	header := abci.Header{}
	return loom.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func TestProcessDeployTx(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:  []byte("myCaller"),
	}

	manager := NewManager()
	manager.Register(VMType_EVM, EvmFactory)
	manager.Register(VMType_PLUGIN, LoomEvmFactory)

	state := mockState()

	evm, err := manager.InitVM(VMType_EVM, state)
	require.Nil(t, err)
	testCryptoZombies(t, evm, caller)
	testLoomTokens(t, evm, caller)

	loomevm, err := manager.InitVM(VMType_PLUGIN, state)
	require.Nil(t, err)
	testCryptoZombies(t, loomevm, caller)
	testLoomTokens(t, loomevm, caller)
}
