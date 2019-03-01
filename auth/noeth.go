// +build !evm

package auth

import (
	"fmt"
)

func verifySolidity(_ SignedTx) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
