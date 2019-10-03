// +build evm

package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain"
)

const (
	LoomPrecompilesStartIndex = 0x20
	MapToLoomAddress          = iota + LoomPrecompilesStartIndex
	MapAddresses
)

func AddLoomPrecompiles(_state loomchain.State, createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error)) {
	index := len(vm.PrecompiledContractsByzantium) + 1

	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(index)})] = &TransferWithBlockchain{}
	index++
	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(index)})] = &TransferPlasmaToken{}

	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(int(MapToLoomAddress))})] = NewMapToLoomAddress(_state, createAddressMapperCtx)
}

type TransferWithBlockchain struct{}

func (t TransferWithBlockchain) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (t TransferWithBlockchain) Run(input []byte) ([]byte, error) {
	strIn := string(input)
	fmt.Println("in TransferWithBlockchain input", strIn)
	return []byte("TransferWithBlockchain"), nil
}

type TransferPlasmaToken struct{}

func (t TransferPlasmaToken) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (t TransferPlasmaToken) Run(input []byte) ([]byte, error) {
	return []byte("TransferPlasmaToken"), nil
}
