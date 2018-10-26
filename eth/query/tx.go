// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
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
	}, nil
}

func GetTxByBlockAndIndex(state loomchain.ReadOnlyState, height, index uint64, readReceipts loomchain.ReadReceiptHandler) (txObj eth.JsonTxObject, err error) {
	params := map[string]interface{}{}
	params["heightPtr"] = &height
	var blockResults *ctypes.ResultBlockResults
	iHeight := int64(height)
	blockResults, err = core.BlockResults(&iHeight)
	if err != nil {
		return txObj, errors.Wrapf(err, "results for block %v", height)
	}
	if len(blockResults.Results.DeliverTx) < int(index) {
		return txObj, errors.Errorf("index %v exceeds size of result array %v", index, len(blockResults.Results.DeliverTx))
	}

	i := uint64(0)
	for _, result := range blockResults.Results.DeliverTx {
		if result.Info == utils.DeployEvm || result.Info == utils.CallEVM {
			if i == index {
				return getTxFromTxResponse(state, *result, readReceipts)
			}
			i++
		}
	}
	return txObj, errors.Errorf("index %v exceeds number of evm transactions %v", index, i)
}

func GetNumEvmTxs(state loomchain.ReadOnlyState, height uint64) (uint64, error) {
	params := map[string]interface{}{}
	params["heightPtr"] = &height
	var blockResults *ctypes.ResultBlockResults
	iHeight := int64(height)
	blockResults, err := core.BlockResults(&iHeight)
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

func getTxFromTxResponse(state loomchain.ReadOnlyState, result abci.ResponseDeliverTx, readReceipts loomchain.ReadReceiptHandler) (txObj eth.JsonTxObject, err error) {
	var txHash []byte
	switch result.Info {
	case utils.DeployEvm:
		dr := vm.DeployResponse{}
		if err := proto.Unmarshal(result.Data, &dr); err != nil {
			return txObj, errors.Wrap(err, "deploy response does not unmarshal")
		}
		drd := vm.DeployResponseData{}
		if err := proto.Unmarshal(dr.Output, &drd); err != nil {
			return txObj, errors.Wrap(err, "deploy response data does not unmarshal")
		}
		txHash = drd.TxHash
	case utils.CallEVM:
		txHash = result.Data
	default:
		return txObj, errors.Errorf("not an EVM transaction")
	}
	return GetTxByHash(state, txHash, readReceipts)
}

func DepreciatedGetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
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
