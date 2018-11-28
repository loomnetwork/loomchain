// +build !evm

package gateway

import (
	"github.com/pkg/errors"
)

type Oracle struct {
}

func CreateLoomCoinOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	return nil, errors.New("not implemented in non-EVM build")
}

func CreateOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	return nil, errors.New("not implemented in non-EVM build")
}

func (orc *Oracle) RunWithRecovery() {
}

func (orc *Oracle) Run() {
}
