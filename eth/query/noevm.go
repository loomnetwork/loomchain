// +build !evm

package query

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func DeprecatedQueryChain(_ string, _ store.BlockStore, _ state.ReadOnlyState,
	_ loomchain.ReadReceiptHandler, _ *evmaux.EvmAuxStore) ([]byte, error) {
	return nil, nil
}

func GetBlockByNumber(_ store.BlockStore, _ state.ReadOnlyState, _ int64, _ bool, _ *evmaux.EvmAuxStore) (eth.JsonBlockObject, error) {
	return eth.JsonBlockObject{}, nil
}

func GetTxObjectFromBlockResult(_ *ctypes.ResultBlock, _ []byte, _ int64, _ *evmaux.EvmAuxStore) (eth.JsonTxObject, *eth.Data, error) {
	return eth.JsonTxObject{}, nil, nil
}

func DeprecatedGetBlockByNumber(
	_ store.BlockStore, _ state.ReadOnlyState, _ int64, _ bool, _ loomchain.ReadReceiptHandler, _ *evmaux.EvmAuxStore,
) ([]byte, error) {
	return nil, nil
}

func GetPendingBlock(_ int64, _ bool, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func DeprecatedGetBlockByHash(
	_ store.BlockStore, _ state.ReadOnlyState, _ []byte, _ bool, _ loomchain.ReadReceiptHandler, _ *evmaux.EvmAuxStore,
) ([]byte, error) {
	return nil, nil
}

func DeprecatedGetTxByHash(_ state.ReadOnlyState, _ []byte, _ loomchain.ReadReceiptHandler) ([]byte, error) {
	return nil, nil
}

func GetBlockHeightFromHash(_ store.BlockStore, _ state.ReadOnlyState, _ []byte) (int64, error) {
	return 0, nil
}

func GetTxByHash(_ state.ReadOnlyState, _ store.BlockStore, _ []byte, _ loomchain.ReadReceiptHandler) (eth.JsonTxObject, error) {
	return eth.JsonTxObject{}, nil
}

func GetTxByBlockAndIndex(_ store.BlockStore, _, _ uint64, _ *evmaux.EvmAuxStore) (txObj eth.JsonTxObject, err error) {
	return eth.JsonTxObject{}, nil
}

func QueryChain(
	_ store.BlockStore, _ state.ReadOnlyState, _ eth.EthFilter, _ loomchain.ReadReceiptHandler, _ *evmaux.EvmAuxStore,
) ([]*types.EthFilterLog, error) {
	return nil, nil
}

func GetNumTxBlock(_ store.BlockStore, _ state.ReadOnlyState, _ int64) (uint64, error) {
	return 0, nil
}
