// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
)

func QueryChain(query string, state loomchain.ReadOnlyState, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockLogs(ethFilter utils.EthBlockFilter, height uint64) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetBlockByNumber(state loomchain.ReadOnlyState, height uint64, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}
