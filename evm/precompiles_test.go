// +build evm

package evm

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/loomnetwork/go-loom"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/vm"
)

func TestMapToLoomAccount(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}
	manager := vm.NewManager()
	manager.Register(vm.VMType_EVM, LoomVmFactory)
	mockState := mockState()
	mockState.SetFeature(features.EvmConstantinopleFeature, true)
	vm, _ := manager.InitVM(vm.VMType_EVM, mockState)
	abiPc, pcAddr := deploySolContract(t, caller, "TestLoomApi", vm)
	_ = abiPc
	_ = pcAddr

	var addrMA common.Address
	addrMA.SetBytes([]byte{byte(int(MapToLoomAddress))})

	local, err := loom.LocalAddressFromHexString("0x5194b63f10691e46635b27925100cfc0a5ceca62")
	require.NoError(t, err)
	msg := append(local, []byte("default")...)

	inputMA, err := abiPc.Pack("TestMappedAccount", addrMA, &msg)
	require.NoError(t, err, "packing parameters")
	fmt.Printf("input length msg %v inputMA hex %x string %s bytes %v\n", len(msg), msg, msg, msg)

	retMA, err := vm.StaticCall(caller, pcAddr, inputMA)
}

func TestPrecompilesAssembly(t *testing.T) {
	caller := loom.Address{
		ChainID: "myChainID",
		Local:   []byte("myCaller"),
	}

	manager := vm.NewManager()
	manager.Register(vm.VMType_EVM, LoomVmFactory)
	mockState := mockState()
	mockState.SetFeature(features.EvmConstantinopleFeature, true)
	vm, _ := manager.InitVM(vm.VMType_EVM, mockState)
	abiPc, pcAddr := deploySolContract(t, caller, "LoomApi", vm)

	numEthPreCompiles := len(ethvm.PrecompiledContractsByzantium)
	msg := []byte("TestInput12345")

	var addr1 common.Address
	addr1.SetBytes([]byte{byte(numEthPreCompiles + 1)})
	input, err := abiPc.Pack("callPFAssembly", addr1, &msg)
	require.NoError(t, err, "packing parameters")
	ret, err := vm.StaticCall(caller, pcAddr, input)
	require.NoError(t, err, "callPFAssembly method on CallPrecompiles")
	st := string(ret)
	_ = st
	fmt.Printf("return TransferWithBlockchain hex %x string %s byes %v\n", ret, ret, ret)
	expected := []byte("TransferWithBlockchain")
	require.True(t, len(expected) <= len(ret))
	actual := ret[:len(expected)]
	_ = actual
	//require.Equal(t, 0, bytes.Compare(expected, actual))

	local, err := loom.LocalAddressFromHexString("0x5194b63f10691e46635b27925100cfc0a5ceca62")
	require.NoError(t, err)
	msg = append(local, []byte("default")...)

	var addrMA common.Address
	addrMA.SetBytes([]byte{byte(int(MapToLoomAddress))})
	inputMA, err := abiPc.Pack("callPFAssembly", addrMA, &msg)
	require.NoError(t, err, "packing parameters")
	fmt.Printf("input length msg %v inputMA hex %x string %s bytes %v\n", len(msg), msg, msg, msg)

	retMA, err := vm.StaticCall(caller, pcAddr, inputMA)

	require.NoError(t, err, "callPFAssembly method on CallPrecompiles")
	fmt.Printf("ma return %x or %s\n", retMA, string(retMA))
}
