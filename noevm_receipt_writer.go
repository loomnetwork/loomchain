// +build !evm

package loomchain

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"

	"github.com/loomnetwork/loomchain/state"
)

type WriteReceiptHandler interface {
	CacheReceipt(
		_ state.State, caller, addr loom.Address, events []*types.EventData, err error, txHash []byte,
	) ([]byte, error)
}
