// +build !evm

package auth

import (
	"fmt"

	"github.com/eosspark/eos-go/crypto/ecc"

	"github.com/loomnetwork/go-loom"
)

func verifySolidity66Byte(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented, non evm build")
}

func verifyTron(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented, non evm build")
}

func verifyEos(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented, non evm build")
}

func verifyEosScatter(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented, non evm build")
}


func LocalAddressFromEosPublicKey(_ ecc.PublicKey) (loom.LocalAddress, error) {
	return nil, fmt.Errorf("not implemented, non evm build")
}