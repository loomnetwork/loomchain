// +build !evm

package auth

import (
	"fmt"
)

func verifySolidity66Byte(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func verifyTron(tx SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("tron support not implemented")
}
