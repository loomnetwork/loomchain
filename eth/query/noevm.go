// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
)

func QueryChain(query string, state loomchain.ReadOnlyState) ([]byte, error) {
	return nil, nil
}

func GetBlockLogs(ethFilter EthBlockFilter, state loomchain.ReadOnlyState, height uint64) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetBlockByNumber(state loomchain.ReadOnlyState, height uint64, full bool) ([]byte, error) {
	return nil, nil
}

func GetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool) ([]byte, error) {
	return nil, nil
}
