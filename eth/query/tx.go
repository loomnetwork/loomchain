// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"

	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

func GetTxByHash(
	blockStore store.BlockStore,
	txHash []byte,
	readReceipts loomchain.ReadReceiptHandler,
	evmAuxStore *evmaux.EvmAuxStore,
) (eth.JsonTxObject, error) {
	txReceipt, err := readReceipts.GetReceipt(txHash)
	if err != nil {
		return eth.GetEmptyTxObject(), errors.Wrap(err, "reading receipt")
	}
	return GetTxByBlockAndIndex(
		blockStore, uint64(txReceipt.BlockNumber),
		uint64(txReceipt.TransactionIndex), evmAuxStore,
	)
}

func GetTxByBlockAndIndex(
	blockStore store.BlockStore, height, index uint64, evmAuxStore *evmaux.EvmAuxStore,
) (eth.JsonTxObject, error) {
	iHeight := int64(height)

	blockResult, err := blockStore.GetBlockByHeight(&iHeight)
	if err != nil {
		return eth.GetEmptyTxObject(), errors.Errorf("error getting block for height %v", height)
	}
	if blockResult == nil || blockResult.Block == nil {
		return eth.GetEmptyTxObject(), errors.Errorf("no block results found at height %v", height)
	}

	if len(blockResult.Block.Data.Txs) <= int(index) {
		return eth.GetEmptyTxObject(), errors.Errorf(
			"tx index out of bounds (%v >= %v)", index, len(blockResult.Block.Data.Txs))
	}

	txResult, err := blockStore.GetTxResult(blockResult.Block.Data.Txs[index].Hash())
	if err != nil {
		return eth.GetEmptyTxObject(), errors.Wrapf(
			err, "failed to find result of tx %X", blockResult.Block.Data.Txs[index].Hash())
	}

	txObj, _, err := GetTxObjectFromBlockResult(blockResult, txResult.TxResult.Data, int64(index), evmAuxStore)
	if err != nil {
		return eth.GetEmptyTxObject(), err
	}
	txObj.TransactionIndex = eth.EncInt(int64(index))

	return txObj, nil
}

func DeprecatedGetTxByHash(
	state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	txReceipt, err := readReceipts.GetReceipt(txHash)
	if err != nil {
		return nil, errors.Wrap(err, "reading receipt")
	}
	caller := loom.UnmarshalAddressPB(txReceipt.CallerAddress)

	txObj := types.EvmTxObject{
		Nonce:    auth.Nonce(state, caller),
		Hash:     txHash,
		Value:    0,
		GasPrice: 0,
		Gas:      0,
		From:     caller.Local,
		To:       txReceipt.ContractAddress,
	}

	if txReceipt.BlockNumber != state.Block().Height {
		txObj.BlockHash = txReceipt.BlockHash
		txObj.BlockNumber = txReceipt.BlockNumber
		txObj.TransactionIndex = 0
	}

	return proto.Marshal(&txObj)
}
