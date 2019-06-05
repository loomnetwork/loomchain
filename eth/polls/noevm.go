// +build !evm

package polls

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
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

func (s *EthSubscriptions) LegacyPoll(_ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(_ string) {
}

func (s EthSubscriptions) Poll(_ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AllLogs(_ loomchain.ReadOnlyState, _ string,
	_ loomchain.ReadReceiptHandler) (interface{}, error) {
	return nil, nil
}

func (s EthSubscriptions) AddLogPoll(_ eth.EthFilter, _ uint64) (string, error) {
	return "", nil
}

func NewEthSubscriptions(_ *evmaux.EvmAuxStore, _ store.BlockStore) *EthSubscriptions {
	return &EthSubscriptions{}
}
