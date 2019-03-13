// +build !evm

package auth

import (
	"fmt"
)

func verifySolidity65Byte(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func verifySolidity66Byte(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
