// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)

func DeprecatedQueryChain(_ string, _ store.BlockStore, _ loomchain.ReadOnlyState, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockByNumber(_ store.BlockStore, _ loomchain.ReadOnlyState, _ int64, _ bool) (eth.JsonBlockObject, error) {
	return eth.JsonBlockObject{}, nil
}

func DeprecatedGetBlockByNumber(
	_ store.BlockStore, _ loomchain.ReadOnlyState, _ int64, _ bool, _ loomchain.ReadReceiptHandler,
) ([]byte, error) {
	return nil, nil
}

func GetPendingBlock(_ int64, _ bool, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DeprecatedGetBlockByHash(
	_ store.BlockStore, _ loomchain.ReadOnlyState, _ []byte, _ bool, _ loomchain.ReadReceiptHandler,
) ([]byte, error) {
	return nil, nil
}

func DeprecatedGetTxByHash(_ loomchain.ReadOnlyState, _ []byte, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockHeightFromHash(_ store.BlockStore, _ loomchain.ReadOnlyState, _ []byte) (int64, error) {
	return 0, nil
}

func GetNumEvmTxBlock(_ store.BlockStore, _ loomchain.ReadOnlyState, _ int64) (uint64, error) {
	return 0, nil
}

func GetTxByHash(_ loomchain.ReadOnlyState, _ []byte, _ loomchain.ReadReceiptHandler) (eth.JsonTxObject, error) {
	return eth.JsonTxObject{}, nil
}

func GetTxByBlockAndIndex(_ store.BlockStore, _ loomchain.ReadOnlyState, _, _ uint64, _ loomchain.ReadReceiptHandler) (txObj eth.JsonTxObject, err error) {
	return eth.JsonTxObject{}, nil
}

func QueryChain(
	_ store.BlockStore, _ loomchain.ReadOnlyState, _ eth.EthFilter, _ loomchain.ReadReceiptHandler,
) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetTxByTendermintHash(_ store.BlockStore, _ []byte) (eth.JsonTxObject, error) {
	return eth.JsonTxObject{}, nil
}

func GetNumTxBlock(_ store.BlockStore, _ loomchain.ReadOnlyState, _ int64) (uint64, error) {
	return 0, nil
}