package rpc

import (
	"encoding/json"
	"math/big"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"

	"github.com/loomnetwork/loomchain/evm/utils"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

const (
	ethChainID = "eth"
	ethTxId = 4
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
	tmTx, err := ethereumToTendermintTx(ethTx)
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

func EthToLoomAddress(ethAddr ecommon.Address) loom.Address {
	return loom.Address{
		ChainID: ethChainID,
		Local:   ethAddr.Bytes(),
	}
}

func ethereumToTendermintTx(txBytes []byte) (ttypes.Tx, error) {
	msg := &vm.MessageTx{}
	msg.Data = txBytes

	var tx types.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		return nil, err
	}

	if tx.To() != nil {
		msg.To = EthToLoomAddress(*tx.To()).MarshalPB()
	}

	chainConfig := utils.DefaultChainConfig()
	ethSigner := types.MakeSigner(&chainConfig, blockNumber)
	from, err := types.Sender(ethSigner, &tx)
	if err != nil {
		return nil, err
	}
	msg.From = EthToLoomAddress(from).MarshalPB()

	txTx := &ltypes.Transaction{
		Id: ethTxId,
	}
	txTx.Data, err = proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	nonceTx := &auth.NonceTx{
		Sequence: tx.Nonce(),
	}
	nonceTx.Inner, err = proto.Marshal(txTx)
	if err != nil {
		return nil, err
	}

	signedTx := &auth.SignedTx{}
	signedTx.Inner, err = proto.Marshal(nonceTx)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(signedTx)
}
