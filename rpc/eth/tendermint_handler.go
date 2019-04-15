package eth

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/tendermint/tendermint/rpc/core"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain/log"
)

type TendermintPRCFunc struct {
	HttpRPCFunc
	name string
}

// Tendermint handlers need parameters translated.
// Only one method supported.
func NewTendermintRPCFunc(funcName string) RPCFunc {
	return &TendermintPRCFunc{
		name: funcName,
	}
}

func (t *TendermintPRCFunc) UnmarshalParamsAndCall(input JsonRpcRequest, conn *websocket.Conn) (json.RawMessage, *Error) {
	var txBytes types.Tx
	switch t.name {
	case "eth_sendRawTransaction":
		var err *Error
		txBytes, err = t.TranslateSendRawTransactionParams(input)
		if err != nil {
			return nil, err
		}
	default:
		return nil, NewError(EcParseError, "Parse parameters", "unknown method")
	}

	r, err := core.BroadcastTxSync(txBytes)
	if err != nil {
		return nil, NewErrorf(EcServer, "Server error", "transaction returned error %v", err)
	}
	if r == nil {
		return nil, NewErrorf(EcServer, "Server error", "transaction returned nil result")
	}

	var result json.RawMessage
	result, err = json.Marshal(EncBytes(r.Hash))
	if err != nil {
		log.Info("marshal transaction hash %v", err)
	}

	return result, nil
}

func (t *TendermintPRCFunc) TranslateSendRawTransactionParams(input JsonRpcRequest) (types.Tx, *Error) {
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
		return nil, NewErrorf(EcParseError, "Parse params", "%v convert input %v to bytes", err, data)
	}

	return types.Tx(txBytes), nil
}
