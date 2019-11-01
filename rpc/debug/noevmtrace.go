// +build !evm

package debug

import (
	"github.com/ethereum/go-ethereum/eth"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

func TraceTransaction(
	_ loomchain.Application,
	_ store.BlockStore,
	_, _, _ int64,
	_ eth.TraceConfig,
) (trace interface{}, err error) {
	return nil, nil
}
