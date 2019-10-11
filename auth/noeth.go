// +build !evm

package auth

import (
	"fmt"

	"github.com/loomnetwork/go-loom/common/evmcompat"
)

func verifySolidity66Byte(_ string, _ SignedTx, _ []evmcompat.SignatureType) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func verifyTron(_ string, _ SignedTx, _ []evmcompat.SignatureType) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func verifyBinance(_ string, _ SignedTx, _ []evmcompat.SignatureType) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func VerifyWrappedEthTx(_ string, signedTx SignedTx, _ []evmcompat.SignatureType) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
