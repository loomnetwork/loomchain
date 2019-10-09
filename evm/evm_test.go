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

	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/store"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	BlockHeight        = int64(34)
	numLoomPreCompiles = 2
)

var (
	PrecompiledRunOutput = ""
	PrecompiledGasOutput = 0
	blockTime            = time.Unix(123456789, 0)
)

func mockState() loomchain.State {
	header := abci.Header{}
	header.Height = BlockHeight
	header.Time = blockTime
	memDB := store.NewMemStore()
	trieDB := trie.NewDatabase(store.NewLoomEthDB(memDB, nil))
	return loomchain.NewStoreState(context.Background(), memDB, header, nil, nil).WithTrieDB(trieDB)
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

// Test that if we add a new precompile, we can call it using the solidity call function.
func TestPrecompiles(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := lvm.NewManager()
	manager.Register(lvm.VMType_EVM, LoomVmFactory)
	state := mockState()
	vm, _ := manager.InitVM(lvm.VMType_EVM, state)
	abiPc, pcAddr := deploySolContract(t, caller, "CallPrecompiles", vm)

	index := len(ethvm.PrecompiledContractsByzantium) + 1
	ethvm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(index)})] = &TestPrecompiledFunction{t: t}

	input, err := abiPc.Pack("callPF", uint32(index), []byte("TestInput"))
	require.NoError(t, err, "packing parameters")
	PrecompiledGasOutput = 0
	PrecompiledRunOutput = ""
	ret, err := vm.StaticCall(caller, pcAddr, input)
	require.Equal(t, 32, len(ret))
	require.Equal(t, byte(1), ret[31], "callPF did not return success")

	require.NoError(t, err, "callPF method on CallPrecompiles")
	require.Equal(t, PrecompiledGasOutput, 123)
	require.Equal(t, PrecompiledRunOutput, "TestPrecompiledFunction")

}

type TestPrecompiledFunction struct {
	t *testing.T
}

func (p TestPrecompiledFunction) RequiredGas(input []byte) uint64 {
	expected := []byte("TestInput")
	require.True(p.t, 0 == bytes.Compare(expected, input[:len(expected)]), "wrong input to required gas")
	PrecompiledGasOutput = 123
	return uint64(0)
}

func (p TestPrecompiledFunction) Run(input []byte) ([]byte, error) {
	expected := []byte("TestInput")
	require.True(p.t, 0 == bytes.Compare(expected, input[:len(expected)]), "wrong input to run")
	PrecompiledRunOutput = "TestPrecompiledFunction"
	return []byte("TestPrecompiledFunction"), nil
}

// Test that we can access the loom precompiles using solidity assembly block
// and return an output value.
func TestPrecompilesAssembly(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := lvm.NewManager()
	manager.Register(lvm.VMType_EVM, LoomVmFactory)
	state := mockState()
	vm, _ := manager.InitVM(lvm.VMType_EVM, state)
	abiPc, pcAddr := deploySolContract(t, caller, "CallPrecompiles", vm)

	numEthPreCompiles := len(ethvm.PrecompiledContractsByzantium)
	AddLoomPrecompiles()
	require.Equal(t, numEthPreCompiles+numLoomPreCompiles, len(ethvm.PrecompiledContractsByzantium))

	msg := []byte("TestInput")
	input, err := abiPc.Pack("callPFAssembly", uint64(numEthPreCompiles+1), &msg)
	require.NoError(t, err, "packing parameters")
	ret, err := vm.StaticCall(caller, pcAddr, input)
	require.NoError(t, err, "callPFAssembly method on CallPrecompiles")
	expected := []byte("TransferWithBlockchain")
	require.True(t, len(expected) <= len(ret))
	actual := ret[:len(expected)]
	require.Equal(t, 0, bytes.Compare(expected, actual))

	input, err = abiPc.Pack("callPFAssembly", uint64(numEthPreCompiles+2), &msg)
	require.NoError(t, err, "packing parameters")
	ret, err = vm.StaticCall(caller, pcAddr, input)
	require.NoError(t, err, "callPFAssembly method on CallPrecompiles")
	expected = []byte("TransferPlasmaToken")
	require.True(t, len(expected) <= len(ret))
	actual = ret[:len(expected)]
	require.Equal(t, 0, bytes.Compare(expected, actual))
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
