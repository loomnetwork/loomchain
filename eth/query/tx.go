// +build evm

package query

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
)

func GetTxByHash(state loomchain.ReadOnlyState, txHash []byte) ([]byte, error) {
	receiptState := store.PrefixKVReader(utils.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	if err != nil {
		return nil, err
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
	/*
		params := map[string]interface{}{} // tx.height=3
		params["query"] = "tx.height >= " + strconv.Itoa(int(txReceipt.BlockNumber)-10) +
			" AND tx.height <= " + strconv.Itoa(int(txReceipt.BlockNumber)+10)
		params["prove"] = false
		params["page"] = 10
		params["perPage"] = 10
		var txResults []*ctypes.ResultTx
		rclient := rpcclient.NewJSONRPCClient(RpcHost)
		_, err = rclient.Call("tx_search", params, &txResults)
	*/
	return proto.Marshal(&txObj)
}

/*
type EvmTxObject struct {
	Hash             []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Nonce            int64  `protobuf:"varint,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	BlockHash        []byte `protobuf:"bytes,3,opt,name=block_hash,json=blockHash,proto3" json:"block_hash,omitempty"`
	BlockNumber      int64  `protobuf:"varint,4,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	TransactionIndex int32  `protobuf:"varint,5,opt,name=transaction_index,json=transactionIndex,proto3" json:"transaction_index,omitempty"`
	From             []byte `protobuf:"bytes,6,opt,name=from,proto3" json:"from,omitempty"`
	To               []byte `protobuf:"bytes,7,opt,name=to,proto3" json:"to,omitempty"`
	Value            int64  `protobuf:"varint,8,opt,name=value,proto3" json:"value,omitempty"`
	GasPrice         int64  `protobuf:"varint,9,opt,name=gas_price,json=gasPrice,proto3" json:"gas_price,omitempty"`
	Gas              int64  `protobuf:"varint,10,opt,name=gas,proto3" json:"gas,omitempty"`
	Input            []byte `protobuf:"bytes,11,opt,name=input,proto3" json:"input,omitempty"`
}*/
