// +build evm

package loomchain

import (
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"

	"github.com/loomnetwork/loomchain/state"
)

type WriteReceiptHandler interface {
	GetEventsFromLogs(
		logs []*eth_types.Log, blockHeight int64, caller, contract loom.Address, input []byte,
	) []*types.EventData
	CacheReceipt(s state.State, caller, addr loom.Address, events []*types.EventData, err error) ([]byte, error)
}
