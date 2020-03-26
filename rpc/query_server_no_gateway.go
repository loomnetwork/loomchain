// +build !gateway

package rpc

import (
	"github.com/pkg/errors"
)

func (s *QueryServer) GetAccountBalances(contract []string) (*AccountsBalanceResponse, error) {
	return nil, errors.New("GetAccountBalances not implemented in this build")
}
