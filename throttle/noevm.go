// +build !evm

package throttle

import (
	"github.com/pkg/errors"
)

func IsEthDeploy(_ []byte) (bool, error) {
	return false, errors.New("ethereum transactions not supported in non evm build")
}
