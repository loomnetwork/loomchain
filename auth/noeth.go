// +build !evm

package auth

import (
	"fmt"
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