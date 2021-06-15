// +build evm

package loomchain

import (
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
)

type WriteReceiptHandler interface {
	GetEventsFromLogs(
		logs []*eth_types.Log, blockHeight int64, caller, contract loom.Address, input []byte,
	) []*types.EventData
	CacheReceipt(
		state State, caller, addr loom.Address, events []*types.EventData, err error, txHash []byte, gasUsed uint64,
	) ([]byte, error)
}
