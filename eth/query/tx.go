// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	`github.com/pkg/errors`
)

func GetTxByHash(state loomchain.ReadOnlyState, txHash []byte, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
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
