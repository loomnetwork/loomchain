package rpc

import (
	"encoding/json"
	"math/big"

	"github.com/gorilla/websocket"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

var (
	blockNumber = big.NewInt(5)
)

type TendermintPRCFunc struct {
	eth.HttpRPCFunc
	name string
	tm   TendermintRpc
}

// Tendermint handlers need parameters translated.
// Only one method supported.
func NewTendermintRPCFunc(funcName string, tm TendermintRpc) eth.RPCFunc {
	return &TendermintPRCFunc{
		name: funcName,
		tm:   tm,
	}
}

func (t *TendermintPRCFunc) UnmarshalParamsAndCall(input eth.JsonRpcRequest, conn *websocket.Conn) (json.RawMessage, *eth.Error) {
	var tmTx ttypes.Tx
	if t.name != "eth_sendRawTransaction" {
		return nil, eth.NewError(eth.EcParseError, "Parse parameters", "unknown method")
	}

	var jErr *eth.Error
	ethTx, jErr := t.TranslateSendRawTransactionParams(input)
	if jErr != nil {
		return nil, jErr
	}
	tmTx, err := t.tm.ethereumToTendermintTx(ethTx)
	if err != nil {
		return nil, eth.NewErrorf(eth.EcServer, "Parse parameters", "convert ethereum tx to tendermint tx, error %v", err)
	}

	r, err := t.tm.BroadcastTxSync(tmTx)
	if err != nil {
		return nil, eth.NewErrorf(eth.EcServer, "Server error", "transaction returned error %v", err)
	}
	if r == nil {
		return nil, eth.NewErrorf(eth.EcServer, "Server error", "transaction returned nil result")
	}

	var result json.RawMessage
	result, err = json.Marshal(eth.EncBytes(r.Hash))
	if err != nil {
		log.Info("marshal transaction hash %v", err)
	}

	return result, nil
}

func (t *TendermintPRCFunc) TranslateSendRawTransactionParams(input eth.JsonRpcRequest) ([]byte, *eth.Error) {
	paramsBytes := []json.RawMessage{}
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return nil, eth.NewError(eth.EcParseError, "Parse params", err.Error())
		}
	}
	var data string
	if err := json.Unmarshal(paramsBytes[0], &data); err != nil {
		return nil, eth.NewErrorf(eth.EcParseError, "Parse params", "unmarshal input parameter err %v", err)
	}

	txBytes, err := eth.DecDataToBytes(eth.Data(data))
	if err != nil {
		return nil, eth.NewErrorf(eth.EcParseError, "Parse params", "%v convert input %v to bytes", err, data)
	}

	return txBytes, nil
}

