// +build !evm

package polls

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
)

type EthSubscriptions struct {
}

func (s EthSubscriptions) DepreciatedAddLogPoll(_ string, _ uint64) (string, error) {
	return "", nil
}

func (s EthSubscriptions) AddBlockPoll(_ uint64) string {
	return ""
}

func (s EthSubscriptions) AddTxPoll(_ uint64) string {
	return ""
}

func (s *EthSubscriptions) DepreciatedPoll(_ loomchain.ReadOnlyState, _ string, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(_ string) {
}

func (s EthSubscriptions) Poll(_ loomchain.ReadOnlyState, _ string, _ loomchain.ReadReceiptHandler) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AllLogs(_ loomchain.ReadOnlyState, _ string, _ loomchain.ReadReceiptHandler) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AddLogPoll(_ utils.EthFilter, _ uint64) (string, error) {
	return "", nil
}

func NewEthSubscriptions() *EthSubscriptions {
	return &EthSubscriptions{}
}
