// +build evm

package evm

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/loomnetwork/go-loom"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/store"
	lvm "github.com/loomnetwork/loomchain/vm"
)

const (
	BlockHeight = int64(34)
)

var (
	blockTime = time.Unix(123456789, 0)
)

func mockState() loomchain.State {
	header := abci.Header{}
	header.Height = BlockHeight
	header.Time = blockTime
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header, nil, nil)
}

func TestProcessDeployTx(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	// Do not test EVM and loomEVM as they do not return a txReceipt
	// so they can be considered to not satisfy the intent of the VM interface.
	// They are also adequately tested by using the loomVM object

	// Test the case where all the transaction are done using one VM
	manager := lvm.NewManager()
	manager.Register(lvm.VMType_PLUGIN, LoomVmFactory)
	loomvm, err := manager.InitVM(lvm.VMType_PLUGIN, mockState())
	require.Nil(t, err)
	testCryptoZombies(t, loomvm, caller)
	testLoomTokens(t, loomvm, caller)

	// Test the case where a new VM is created for each transaction, EVM changes
	// committed to the state.
	// The state carries over to be used to create the VM for the next transaction.
	testCryptoZombiesUpdateState(t, mockState(), caller)
}

func TestValue(t *testing.T) {
	const negativeNumber = -34
	const positiveNumber = 24

	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}
	manager := lvm.NewManager()
	manager.Register(lvm.VMType_EVM, LoomVmFactory)
	state := mockState()

	vm, _ := manager.InitVM(lvm.VMType_EVM, state)

	testValue(t, state, vm, caller, true, negativeNumber)
	testValue(t, state, vm, caller, true, positiveNumber)
	testValue(t, state, vm, caller, false, negativeNumber)
	testValue(t, state, vm, caller, false, positiveNumber)

}

func testValue(t *testing.T, state loomchain.State, vm lvm.VM, caller loom.Address, checkTxValueFeature bool, value int64) {
	defer func() {
		if r := recover(); r != nil {
			require.True(t, !checkTxValueFeature && value < 0)
		}
	}()

	bytetext, err := ioutil.ReadFile("testdata/GlobalProperties.bin")
	require.NoError(t, err, "reading "+"GlobalProperties"+".bin")
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")

	state.SetFeature(features.CheckTxValueFeature, checkTxValueFeature)
	_, _, err = vm.Create(caller, bytecode, loom.NewBigUIntFromInt(value))
	if checkTxValueFeature && value < 0 {
		require.Error(t, err)
		require.Equal(t, err.Error(), fmt.Sprintf("value %v must be non negative", big.NewInt(value)))
	}
}

// This tests that the Solidity global variables match the corresponding
// values set in the vm.EVM object.
// Only tests where we have specifically set non-default values.
func TestGlobals(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := lvm.NewManager()
	manager.Register(lvm.VMType_EVM, LoomVmFactory)
	state := mockState()
	vm, _ := manager.InitVM(lvm.VMType_EVM, state)
	abiGP, gPAddr := deploySolContract(t, caller, "GlobalProperties", vm)

	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testNow(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testBlockTimeStamp(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testBlockNumber(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testTxOrigin(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testMsgSender(t, abiGP, caller, gPAddr, vm)
	vm, _ = manager.InitVM(lvm.VMType_EVM, state)
	testMsgValue(t, abiGP, caller, gPAddr, vm)
}

func testMsgValue(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("msgValue")
	require.NoError(t, err, "packing parameters")
	_, err = vm.Call(caller, gPAddr, input, loom.NewBigUIntFromInt(7))
	require.Equal(t, "insufficient balance for transfer", err.Error())

	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling msgValue method on GlobalProperties")
	var actual *big.Int
	require.NoError(t, abiGP.Unpack(&actual, "msgValue", res), "unpacking result of call to msgValue")
	require.Equal(t, int64(0), actual.Int64(), "wrong value returned for msgValue")
}

func testNow(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("Now")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling Now method on GlobalProperties")
	var now *big.Int
	require.NoError(t, abiGP.Unpack(&now, "Now", res), "unpacking result of call to Now")
	require.Equal(t, blockTime.Unix(), now.Int64(), "wrong value returned for Now")
}

func testBlockTimeStamp(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("blockTimeStamp")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling blockTimeStamp method on GlobalProperties")
	var actual *big.Int
	require.NoError(t, abiGP.Unpack(&actual, "blockTimeStamp", res), "unpacking result of call to blockTimeStamp")
	require.Equal(t, blockTime.Unix(), actual.Int64(), "wrong value returned for blockTimeStamp")
}

func testBlockNumber(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("blockNumber")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling blockNumber method on GlobalProperties")
	var actual *big.Int
	require.NoError(t, abiGP.Unpack(&actual, "blockNumber", res), "unpacking result of call to blockNumber")
	require.Equal(t, BlockHeight, actual.Int64(), "wrong value returned for blockNumber")
}

func testTxOrigin(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("txOrigin")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling txOrigin method on GlobalProperties")
	require.True(t, len(res) >= len(caller.Local), "returned address too short")
	actual := res[len(res)-len(caller.Local):]
	expected := caller.Local
	require.True(
		t,
		bytes.Equal(actual, expected),
		"returned address should match caller",
	)
}

func testMsgSender(t *testing.T, abiGP abi.ABI, caller, gPAddr loom.Address, vm lvm.VM) {
	input, err := abiGP.Pack("msgSender")
	require.NoError(t, err, "packing parameters")
	res, err := vm.StaticCall(caller, gPAddr, input)
	require.NoError(t, err, "calling msgSender method on GlobalProperties")
	require.True(t, len(res) >= len(caller.Local), "returned address too short")
	actual := res[len(res)-len(caller.Local):]
	expected := caller.Local
	require.True(
		t,
		bytes.Equal(actual, expected),
		"returned address should match caller",
	)
}

func deploySolContract(t *testing.T, caller loom.Address, filename string, vm lvm.VM) (abi.ABI, loom.Address) {
	bytetext, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	require.NoError(t, err, "reading "+filename+".bin")
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")

	_, addr, err := vm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))

	require.NoError(t, err, "deploying "+filename+" on EVM")
	simpleStoreData, err := ioutil.ReadFile("testdata/" + filename + ".abi")
	require.NoError(t, err, "reading "+filename+".abi")
	ethAbi, err := abi.JSON(strings.NewReader(string(simpleStoreData)))
	require.NoError(t, err, "reading abi")
	return ethAbi, addr
}
