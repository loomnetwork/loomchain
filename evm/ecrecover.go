package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ecrecoverAddr = common.BytesToAddress([]byte{1})
)

func customLoomPrecompiles() {
	vm.PrecompiledContractsByzantium[ecrecoverAddr] = &ecrecover{}
	vm.PrecompiledContractsHomestead[ecrecoverAddr] = &ecrecover{}
}

type ecrecover struct {
}

func (p ecrecover) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (p ecrecover) Run(input []byte) ([]byte, error) {
	const ecRecoverInputLength = 128
	hashBytes := input[32:64]
	_ = hashBytes
	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	hashBytes = input[0:32]
	_ = hashBytes
	rBytes := input[64:96]
	_ = rBytes
	sBytes := input[96:128]
	_ = sBytes

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], append(input[64:128], v))
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

func allZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}
