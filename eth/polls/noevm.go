// +build !evm

package polls

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

type EthSubscriptions struct {
}

func (s EthSubscriptions) AddLogPoll(filter string, height uint64) (string, error) {
	return "", nil
}

func (s EthSubscriptions) AddBlockPoll(height uint64) string {
	return ""
}

func (s EthSubscriptions) AddTxPoll(height uint64) string {
	return ""
}

func (s *EthSubscriptions) Poll(
	blockStore store.BlockStore, state loomchain.ReadOnlyState, id string,
	readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(id string) {
}

func NewEthSubscriptions() *EthSubscriptions {
	return &EthSubscriptions{}
}
