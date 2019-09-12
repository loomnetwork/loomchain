// +build !evm

package loomchain

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
)

type WriteReceiptHandler interface {
	CacheReceipt(
		state State, caller, addr loom.Address, events []*types.EventData, err error,
	) ([]byte, error)
}
