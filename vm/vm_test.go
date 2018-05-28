// +build evm

package vm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

func mockState() loomchain.State {
	header := abci.Header{}
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func TestProcessDeployTx(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := NewManager()
	manager.Register(VMType_EVM, EvmFactory)
	manager.Register(VMType_PLUGIN, LoomEvmFactory)

	evm, err := manager.InitVM(VMType_EVM, mockState())
	require.Nil(t, err)
	testCryptoZombies(t, evm, caller)
	testLoomTokens(t, evm, caller)

	loomevm, err := manager.InitVM(VMType_PLUGIN, mockState())
	require.Nil(t, err)
	testCryptoZombies(t, loomevm, caller)
	testLoomTokens(t, loomevm, caller)

	manager.Register(VMType_PLUGIN, LoomVmFactory)
	loomvm, err := manager.InitVM(VMType_PLUGIN, mockState())
	require.Nil(t, err)
	testCryptoZombies(t, loomvm, caller)
	testLoomTokens(t, loomvm, caller)

	testCryptoZombiesUpdateState(t, mockState(), caller)
}
