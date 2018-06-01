// +build evm

package vm

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"
)

const (
	BlockHeight = int64(34)
	BlockTime   = int64(123456789)
)

func mockState() loomchain.State {
	header := abci.Header{}
	header.Height = BlockHeight
	header.Time = BlockTime
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

// This tests that the Solidity global variables match the corresponding
// values set in the vm.EVM object.
// Only tests where we have specifically set non-default values.
func TestGlobals(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := NewManager()
	manager.Register(VMType_EVM, LoomVmFactory)
	state := mockState()

	bytetext, err := ioutil.ReadFile("testdata/GlobalProperties.bin")
	require.NoError(t, err, "reading GlobalProperties.bin")
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")
	vm, _ := manager.InitVM(VMType_EVM, state)
	_, gPAddr, err := vm.Create(caller, bytecode)
	require.NoError(t, err, "deploying GlobalPropertiy on EVM")
	simpleStoreData, err := ioutil.ReadFile("testdata/GlobalProperties.abi")
	require.NoError(t, err, "reading GlobalProperties.abi")
	abiGP, err := abi.JSON(strings.NewReader(string(simpleStoreData)))
	require.NoError(t, err, "reading abi")

	vm, _ = manager.InitVM(VMType_EVM, state)
	testNow(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(VMType_EVM, state)
	testBlockTimeStamp(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(VMType_EVM, state)
	testBlockNumber(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(VMType_EVM, state)
	testTxOrigin(t, abiGP, caller, gPAddr, vm)

}

func testNow(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm VM) {
	input, err := abiGP.Pack("Now")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling Now method on GlobalProperties")
	var now *big.Int
	require.NoError(t, abiGP.Unpack(&now, "Now", res), "unpacking result of call to Now")
	require.Equal(t, BlockTime, now.Int64(), "wrong value returned for Now")
}

func testBlockTimeStamp(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm VM) {
	input, err := abiGP.Pack("blockTimeStamp")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling blockTimeStamp method on GlobalProperties")
	var actual *big.Int
	require.NoError(t, abiGP.Unpack(&actual, "blockTimeStamp", res), "unpacking result of call to blockTimeStamp")
	require.Equal(t, BlockTime, actual.Int64(), "wrong value returned for blockTimeStamp")
}

func testBlockNumber(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm VM) {
	input, err := abiGP.Pack("blockNumber")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling blockNumber method on GlobalProperties")
	var actual *big.Int
	require.NoError(t, abiGP.Unpack(&actual, "blockNumber", res), "unpacking result of call to blockNumber")
	require.Equal(t, BlockHeight, actual.Int64(), "wrong value returned for blockNumber")
}

func testTxOrigin(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm VM) {
	input, err := abiGP.Pack("txOrigin")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling txOrigin method on GlobalProperties")
	require.True(t, len(res) >= len(caller.Local), "returned address too short")
	actual := res[len(res)-len(caller.Local):]
	expected := caller.Local
	require.True(
		t,
		bytes.Compare(actual, expected) == 0,
		"returned address should match caller",
	)
}
