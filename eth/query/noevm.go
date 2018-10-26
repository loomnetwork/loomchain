// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	rpcutils "github.com/loomnetwork/loomchain/rpc/eth"
)

func QueryChain(query string, state loomchain.ReadOnlyState, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockLogs(ethFilter utils.EthBlockFilter, height uint64) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetBlockByNumber(state loomchain.ReadOnlyState, height int64, full bool, readReceipts loomchain.ReadReceiptHandler) (rpcutils.JsonBlockObject, error) {
	return rpcutils.JsonBlockObject{}, nil
}

func DepreciatedGetBlockByNumber(state loomchain.ReadOnlyState, height int64, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetPendingBlock(height int64, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DepreciatedGetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DepreciatedGetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}
