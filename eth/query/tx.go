// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"

	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/common"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)

func GetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) (eth.JsonTxObject, error) {
	txReceipt, err := readReceipts.GetReceipt(state, txHash)
	if err != nil {
		return eth.JsonTxObject{}, errors.Wrap(err, "reading receipt")
	}
	return eth.JsonTxObject{
		Nonce:            eth.EncInt(txReceipt.Nonce),
		Hash:             eth.EncBytes(txHash),
		BlockHash:        eth.EncBytes(txReceipt.BlockHash),
		BlockNumber:      eth.EncInt(txReceipt.BlockNumber),
		TransactionIndex: eth.EncInt(int64(txReceipt.TransactionIndex)),
		From:             eth.EncAddress(txReceipt.CallerAddress),
		To:               eth.EncBytes(txReceipt.ContractAddress),

		Gas:      eth.EncInt(0),
		GasPrice: eth.EncInt(0),
		Input:    "0x0", //todo investigate adding input
		Value:    eth.EncInt(0),
	}, nil
}

func GetTxByTendermintHash(blockStore store.BlockStore, hash []byte) (eth.JsonTxObject, error) {
	txResults, err := blockStore.GetTxResult(hash)
	if err != nil {
		return eth.JsonTxObject{}, err
	}
	blockResult, err := blockStore.GetBlockByHeight(&txResults.Height)
	if err != nil {
		return eth.JsonTxObject{}, err
	}
	return GetTxObjectFromBlockResult(blockResult, int64(txResults.Index))
}

func GetTxByBlockAndIndex(
	state loomchain.State,
	blockStore store.BlockStore,
	height,
	index uint64,
) (txObj eth.JsonTxObject, err error) {
	iHeight := int64(height)

	blockResult, err := blockStore.GetBlockByHeight(&iHeight)
	if blockResult == nil || blockResult.Block == nil {
		return txObj, errors.Errorf("no block results found at height %v", height)
	}

	if len(blockResult.Block.Data.Txs) <= int(index) {
		return txObj, errors.Errorf("tx index out of bounds (%v >= %v)", index, len(blockResult.Block.Data.Txs))
	}

	txObj, err = GetTxObjectFromBlockResult(blockResult, int64(index))
	if err != nil {
		return txObj, err
	}
	txObj.TransactionIndex = eth.EncInt(int64(index))

	txResult, err := blockStore.GetTxResult(blockResult.Block.Data.Txs[index].Hash())
	if txResult == nil {
		return txObj, errors.Wrapf(err, "cant find tx result, hash %X", blockResult.Block.Data.Txs[index].Hash())
	}
	if txResult.TxResult.Info == utils.CallEVM || txResult.TxResult.Info == utils.DeployEvm {
		txHashList, err := common.GetTxHashList(state, uint64(height))
		if err != nil {
			return txObj, errors.Wrapf(err, "tx-hahs list at height %v", height)
		}
		evmIndex := 0
		for i := uint64(0); i <= index; i++ {
			tx := blockResult.Block.Data.Txs[i]
			txResult, err := blockStore.GetTxResult(tx.Hash())
			if err != nil {
				return txObj, errors.Wrapf(err, "cant find tx details, hash %X", tx.Hash())
			}
			if txResult.TxResult.Info == utils.CallEVM || txResult.TxResult.Info == utils.DeployEvm {
				evmIndex++
			}
		}
		txObj.Hash = eth.EncBytes(txHashList[evmIndex])
	}

	return txObj, nil
}

func DeprecatedGetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	txReceipt, err := readReceipts.GetReceipt(state, txHash)
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
