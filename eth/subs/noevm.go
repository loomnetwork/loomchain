// +build !evm

package subs

import (
	"github.com/loomnetwork/loomchain"
)

type EthSubscriptions struct {
}

func (s *EthSubscriptions) Add(filter string) (string, error) {
	return "", nil
}

func (s *EthSubscriptions) Poll(state loomchain.ReadOnlyState, id string) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(id string) {
}

func NewEthSubscriptions() *EthSubscriptions {
	return &EthSubscriptions{}
}
