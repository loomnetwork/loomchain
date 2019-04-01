package eth

import (
	"encoding/json"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain/log"
)

const (
	DeployEvm = "deploy.evm"
	CallEVM   = "call.evm"
)

type TendermintPRCFunc struct {
	name string
}

// Tendermint handlers need parameters translated.
// Only one method supported.
func NewTendermintRPCFunc(funcName string) RPCFunc {
	return &TendermintPRCFunc{
		name: funcName,
	}
}

func (t TendermintPRCFunc) unmarshalParamsAndCall(input JsonRpcRequest, writer http.ResponseWriter, reader *http.Request, conn *websocket.Conn) (*JsonRpcResponse, *Error) {
	var txBytes types.Tx

	switch t.name {
	case "eth_sendRawTransaction":
		var err *Error
		txBytes, err = t.TranslateSendRawTransactionParmas(input)
		if err != nil {
			return nil, err
		}
	default:
		return nil, NewError(EcParseError, "Parse parameters", "unknown method")
	}

	r, err := core.BroadcastTxCommit(txBytes)
	if err != nil {
		return nil, NewErrorf(EcServer, "Server error", "transaction returned error %v", err)
	}
	if r == nil {
		return nil, NewErrorf(EcServer, "Server error", "transaction returned nil result")
	}
	if r.CheckTx.Code != abci.CodeTypeOK {
		return nil, NewErrorf(EcServer, "Server error", "transaction failed %v", r.CheckTx.Log)
	}
	if r.DeliverTx.Code != abci.CodeTypeOK {
		return nil, NewErrorf(EcServer, "Server error", "transaction failed %v", r.DeliverTx.Log)
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
