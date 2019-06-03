// +build !evm

package auth

import (
	"fmt"
)

func VerifySolidity66Byte(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func verifyTron(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
