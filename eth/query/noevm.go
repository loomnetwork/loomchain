// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts"
)

func QueryChain(query string, state loomchain.ReadOnlyState, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockLogs(ethFilter utils.EthBlockFilter, height uint64) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetBlockByNumber(state loomchain.ReadOnlyState, height uint64, full bool, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}
