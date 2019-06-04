// +build !evm

package polls

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)

type EthSubscriptions struct {
}

func (s EthSubscriptions) LegacyAddLogPoll(_ string, _ uint64) (string, error) {
	return "", nil
}

func (s EthSubscriptions) AddBlockPoll(_ uint64) string {
	return ""
}

func (s EthSubscriptions) AddTxPoll(_ uint64) string {
	return ""
}

func (s *EthSubscriptions) LegacyPoll(_ store.BlockStore, _ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler, _ *store.EvmAuxStore) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(_ string) {
}

func (s EthSubscriptions) Poll(_ store.BlockStore, _ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler, _ *store.EvmAuxStore) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AllLogs(_ store.BlockStore, _ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler, _ *store.EvmAuxStore) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AddLogPoll(_ eth.EthFilter, _ uint64, _ *store.EvmAuxStore) (string, error) {
	return "", nil
}

func NewEthSubscriptions() *EthSubscriptions {
	return &EthSubscriptions{}
}
