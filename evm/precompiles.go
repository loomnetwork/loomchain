// +build evm

package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func AddLoomPrecompiles() {
	index := len(vm.PrecompiledContractsByzantium) + 1
	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(index)})] = &TransferWithBlockchain{}
	index++
	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(index)})] = &TransferPlasmaToken{}
}

type TransferWithBlockchain struct{}

func (t TransferWithBlockchain) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (t TransferWithBlockchain) Run(input []byte) ([]byte, error) {
	return []byte("W"), nil
}

type TransferPlasmaToken struct{}

func (t TransferPlasmaToken) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (t TransferPlasmaToken) Run(input []byte) ([]byte, error) {
	return []byte("P"), nil
}
