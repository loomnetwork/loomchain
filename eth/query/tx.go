// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
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
	return eth.JsonTxObject{
		Nonce:                  eth.ZeroedQuantity,
		Hash:                   eth.EncBytes(hash),
		BlockHash:              eth.EncBytes(blockResult.BlockMeta.BlockID.Hash),
		BlockNumber:            eth.EncInt(txResults.Height),
		TransactionIndex:       "0x0",
		From:                   eth.ZeroedData20Bytes,
		To:                     eth.ZeroedData20Bytes,
		Gas:                    eth.EncInt(txResults.TxResult.GasWanted),
		GasPrice:               eth.EncInt(txResults.TxResult.GasUsed),
		Input:                  eth.EncBytes(txResults.Tx),
		Value:                  eth.EncInt(0),
	}, nil
}

func GetTxByBlockAndIndex(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	height,
	index uint64,
	readReceipts loomchain.ReadReceiptHandler,
) (txObj eth.JsonTxObject, err error) {
	iHeight := int64(height)

	blockResult, err := blockStore.GetBlockByHeight(&iHeight)
	if blockResult == nil || blockResult.Block == nil {
		return txObj, errors.Errorf("no block results found at height %v", height)
	}
	if len(blockResult.Block.Data.Txs) <= int(index) {
		return txObj, errors.Errorf("tx index out of bounds (%v >= %v)", index, len(blockResult.Block.Data.Txs))
	}

	txResult, err := blockStore.GetTxResult(blockResult.Block.Data.Txs[index].Hash())
	if err != nil {
		return txObj, errors.Errorf("no tx with hash %v found in block %v", blockResult.Block.Data.Txs[index].Hash(), height)
	}
	txObj, err = getTxFromTxResponse(state, txResult, blockResult,  readReceipts)
	if err != nil {
		return txObj, err
	}
	txObj.TransactionIndex = eth.EncInt(int64(index))
	return txObj, nil
}

func GetNumEvmTxs(blockStore store.BlockStore, state loomchain.ReadOnlyState, height uint64) (uint64, error) {
	params := map[string]interface{}{}
	params["heightPtr"] = &height
	var blockResults *ctypes.ResultBlockResults
	iHeight := int64(height)
	blockResults, err := blockStore.GetBlockResults(&iHeight)
	if err != nil {
		return 0, errors.Wrapf(err, "results for block %v", height)
	}

	count := uint64(0)
	for _, result := range blockResults.Results.DeliverTx {
		if result.Info == utils.DeployEvm || result.Info == utils.CallEVM {
			count++
		}
	}
	return count, nil
}

func getTxFromTxResponse(state loomchain.ReadOnlyState, txResult *ctypes.ResultTx, blockResult *ctypes.ResultBlock, readReceipts loomchain.ReadReceiptHandler) (txObj eth.JsonTxObject, err error) {
	switch txResult.TxResult.Info {
	case utils.DeployEvm:
		dr := vm.DeployResponse{}
		if err := proto.Unmarshal(txResult.TxResult.Data, &dr); err != nil {
			return txObj, errors.Wrap(err, "deploy response does not unmarshal")
		}
		drd := vm.DeployResponseData{}
		if err := proto.Unmarshal(dr.Output, &drd); err != nil {
			return txObj, errors.Wrap(err, "deploy response data does not unmarshal")
		}
		return GetTxByHash(state, drd.TxHash, readReceipts)
	case utils.CallEVM:
		return GetTxByHash(state, txResult.TxResult.Data, readReceipts)
	case utils.CallPlugin:
		fallthrough
	case utils.DeployPlugin:
		return eth.JsonTxObject{
			Nonce:                  eth.ZeroedQuantity,
			Hash:                   eth.EncBytes(txResult.Hash),
			BlockHash:              eth.EncBytes(blockResult.BlockMeta.BlockID.Hash),
			BlockNumber:            eth.EncInt(txResult.Height),
			From:                   eth.ZeroedData20Bytes,
			To:                     eth.ZeroedData20Bytes,
			Gas:                    eth.EncInt(txResult.TxResult.GasWanted),
			GasPrice:               eth.EncInt(txResult.TxResult.GasUsed),
			Input:                  eth.EncBytes(txResult.Tx),
			Value:                  eth.EncInt(0),
		}, nil
	default:
		return txObj, errors.Errorf("unknown transaction type %v", txResult.TxResult.Info)
	}
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
