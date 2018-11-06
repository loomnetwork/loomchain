// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

func DepreciatedQueryChain(_ string, _ loomchain.ReadOnlyState, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockByNumber(_ loomchain.ReadOnlyState, _ int64, _ bool, _ loomchain.ReadReceiptHandler) (eth.JsonBlockObject, error) {
	return eth.JsonBlockObject{}, nil
}

func DepreciatedGetBlockByNumber(_ loomchain.ReadOnlyState, _ int64, _ bool, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetPendingBlock(_ int64, _ bool, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DepreciatedGetBlockByHash(_ loomchain.ReadOnlyState, _ []byte, _ bool, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DepreciatedGetTxByHash(_ loomchain.ReadOnlyState, _ []byte, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockHeightFromHash(_ loomchain.ReadOnlyState, _ []byte) (int64, error) {
	return 0, nil
}

func GetNumEvmTxBlock(_ loomchain.ReadOnlyState, _ int64) (uint64, error) {
	return 0, nil
}

func GetTxByHash(_ loomchain.ReadOnlyState, _ []byte, _ loomchain.ReadReceiptHandler) (eth.JsonTxObject, error) {
	return eth.JsonTxObject{}, nil
}

func GetTxByBlockAndIndex(_ loomchain.ReadOnlyState, _, _ uint64, _ loomchain.ReadReceiptHandler) (txObj eth.JsonTxObject, err error) {
	return eth.JsonTxObject{}, nil
}

func QueryChain(_ loomchain.ReadOnlyState, _ eth.EthFilter, _ loomchain.ReadReceiptHandler) ([]*types.EthFilterLog, error) {
	return nil, nil
}
