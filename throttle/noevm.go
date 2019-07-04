// +build !evm

package throttle

import (
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
)

func isEthDeploy(_ []byte) (bool, error) {
	return false, errors.New("ethereum transactions not supported in non evm build")
}

func ethTxBytes(_ uint64, _ loom.Address, _ []byte) ([]byte, error) {
	return nil, errors.New("ethereum transactions not supported in non evm build")
}