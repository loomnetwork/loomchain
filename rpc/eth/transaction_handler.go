package eth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/log"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom/vm"
)

const (
	//testTx = "0a90050a8b0508011286050a00121f0a0764656661756c7412149a1ac42a17aad6dbc6d21c162989d0f7010740441ae004080112ce04608060405234801561001057600080fd5b5061022e806100206000396000f3fe608060405234801561001057600080fd5b506004361061005e576000357c01000000000000000000000000000000000000000000000000000000009004806360fe47b1146100635780636d4ce63c14610091578063cf718921146100e2575b600080fd5b61008f6004803603602081101561007957600080fd5b8101908080359060200190929190505050610110565b005b610099610180565b604051808381526020018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019250505060405180910390f35b61010e600480360360208110156100f857600080fd5b8101908080359060200190929190505050610192565b005b806000819055506000547f7e0b7a35f017ec94e71d7012fe8fa8011f1dab6090674f92de08f8092ab30dda33604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a250565b60008060005433809050915091509091565b806000819055506000547fbd0b1e25f4b9c4b15621999967b6f720a9d31b208d1b70ec690fb4f46d445c8233604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a25056fea165627a7a72305820eab32966bdd60595bc33cf239219f1b6804d6cd7287d7e9339dd539078103efc00291a0b53696d706c6553746f72651003124036ccdb9aa0cb1144fceb0a71679a7fbf4fa35531ab4485a69820e5534ae28ba32e4d2c13584cd3751807091c0f15e912c1c09e7b70adb387a67f1d468eaf89001a20f60c7e22684970ea51f6017c8b8add4fb614b515e816ae1b40afe4d2d03779e7"
	//nilHash = "0x00000000000000000000"
	DeployEvm = "deploy.evm"
	CallEVM   = "call.evm"
)

type TendermintPRCFunc struct {
	HttpRPCFunc
	name string
}

func NewTendermintRPCFunc(method interface{}, paramNamesString, funcName string) RPCFunc {
	var paramNames []string
	if len(paramNamesString) > 0 {
		paramNames = strings.Split(paramNamesString, ",")

	} else {
		paramNames = []string{}
	}

	rMethod := reflect.TypeOf(method)
	if len(paramNames) != rMethod.NumIn() {
		panic("parameter count mismatch making loom api method")
	}
	signature := []reflect.Type{}
	for p := 0; p < rMethod.NumIn(); p++ {
		signature = append(signature, rMethod.In(p))
	}

	return &TendermintPRCFunc{
		HttpRPCFunc: HttpRPCFunc{
			method:    reflect.ValueOf(method),
			signature: signature,
		},
		name: funcName,
	}

}

func (t TendermintPRCFunc) unmarshalParamsAndCall(input JsonRpcRequest, writer http.ResponseWriter, reader *http.Request, conn *websocket.Conn) (resp *JsonRpcResponse, jsonErr *Error) {
	var err error
	var txBytes types.Tx

	switch t.name {
	case "eth_sendRawTransaction":
		txBytes, err = t.TranslateSendRawTransactionParmas(input)
	case "eth_sendTransaction":
		txBytes, err = t.TranslateSendTransactionParmas(input)
	default:
		err = fmt.Errorf("unknown method %v", t.name)
	}

	r, err := core.BroadcastTxCommit(txBytes)
	if err != nil {
		return resp, NewErrorf(EcServer, "Server error", "transaction returned error %v", err)
	}
	if r == nil {
		return resp, NewErrorf(EcServer, "Server error", "transaction returned nil result")
	}
	if r.CheckTx.Code != abci.CodeTypeOK {
		return resp, NewErrorf(EcServer, "Server error", "transaction failed %v", r.CheckTx.Log)
	}
	if r.DeliverTx.Code != abci.CodeTypeOK {
		return resp, NewErrorf(EcServer, "Server error", "transaction failed %v", r.DeliverTx.Log)
	}

	result, err := json.Marshal(EncBytes(getHashFromResult(r)))
	if err != nil {
		log.Info("marshal transaction hash %v", err)
	}

	return &JsonRpcResponse{
		Result:  result,
		Version: "2.0",
		ID:      input.ID,
	}, nil
}

func (t TendermintPRCFunc) TranslateSendRawTransactionParmas(input JsonRpcRequest) (types.Tx, *Error) {
	paramsBytes := []json.RawMessage{}
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return nil, NewError(EcParseError, "Parse params", err.Error())
		}
	}
	var data string
	if err := json.Unmarshal(paramsBytes[0], &data); err != nil {
		return nil, NewErrorf(EcParseError, "Parse params", "unmarshal input parameter err %v", err)
	}

	txBytes, err := DecDataToBytes(Data(data))
	if err != nil {
		return nil, NewErrorf(EcParseError, "Parse params", "%v convert input %v to bytes %v", err, data)
	}

	return types.Tx(txBytes), nil
}

func (t TendermintPRCFunc) TranslateSendTransactionParmas(input JsonRpcRequest) (types.Tx, *Error) {
	return nil, NewErrorf(EcParseError, "Parse params", "eth_sendTransaction not implemented")
}

func getHashFromResult(r *ctypes.ResultBroadcastTxCommit) []byte {
	if r.DeliverTx.Info == CallEVM {
		return r.DeliverTx.Data
	}
	if r.DeliverTx.Info == DeployEvm {
		response := vm.DeployResponse{}
		if err := proto.Unmarshal(r.DeliverTx.Data, &response); err != nil {
			log.Info("%v unmarshal transaction deploy-response %v", err, r.DeliverTx.Data)
			return nil
		}
		output := vm.DeployResponseData{}
		if err := proto.Unmarshal(response.Output, &output); err != nil {
			log.Info("%v unmarshal transaction deploy-response-data %v", err, response.Output)
			return nil
		}
		return output.TxHash
	}
	return nil
}

/**/
