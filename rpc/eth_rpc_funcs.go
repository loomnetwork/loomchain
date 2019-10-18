package rpc

import (
	"encoding/json"
	"fmt"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

// TODO: This doesn't belong in this package, should be in rpc/eth, but moving it there is non-trivial
//       due to import cycles
type SendRawTransactionPRCFunc struct {
	eth.HttpRPCFunc
	chainID     string
	ethSigner   etypes.Signer
	broadcastTx func(tx types.Tx) (*ctypes.ResultBroadcastTx, error)
}

// Tendermint handlers need parameters translated.
func NewSendRawTransactionRPCFunc(chainID string, broadcastTx func(tx types.Tx) (*ctypes.ResultBroadcastTx, error)) eth.RPCFunc {
	ethChainID, err := evmcompat.ToEthereumChainID(chainID)
	if err != nil {
		panic(err)
	}
	return &SendRawTransactionPRCFunc{
		chainID:     chainID,
		ethSigner:   etypes.NewEIP155Signer(ethChainID),
		broadcastTx: broadcastTx,
	}
}

// UnmarshalParamsAndCall implements RPCFunc
func (t *SendRawTransactionPRCFunc) UnmarshalParamsAndCall(
	input eth.JsonRpcRequest, conn *websocket.Conn,
) (json.RawMessage, *eth.Error) {
	if len(input.Params) == 0 {
		return nil, eth.NewError(eth.EcInvalidParams, "Parse params", "expected one or more parameters")
	}

	paramsBytes := []json.RawMessage{}
	if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
		return nil, eth.NewError(eth.EcParseError, "Parse params", err.Error())
	}

	var data string
	if err := json.Unmarshal(paramsBytes[0], &data); err != nil {
		return nil, eth.NewErrorf(eth.EcParseError, "Parse params", "failed to unmarshal input: %v", err)
	}

	txBytes, err := eth.DecDataToBytes(eth.Data(data))
	if err != nil {
		return nil, eth.NewErrorf(
			eth.EcParseError, "Parse params", "failed to convert input %v to bytes: %v", data, err,
		)
	}

	tmTx, err := t.ethereumToTendermintTx(txBytes)
	if err != nil {
		return nil, eth.NewError(
			eth.EcServer, fmt.Sprintf("failed to wrap eth tx: %v", err), "",
		)
	}

	r, err := t.broadcastTx(tmTx)
	if err != nil {
		return nil, eth.NewErrorf(eth.EcServer, "Server error", "transaction returned error %v", err)
	}
	if r == nil {
		return nil, eth.NewError(eth.EcServer, "Server error", "transaction returned nil result")
	}
	if r.Code != 0 {
		return nil, eth.NewError(eth.EcServer, fmt.Sprintf("CheckTx failed: %s", r.Log), "")
	}

	var result json.RawMessage
	result, err = json.Marshal(eth.EncBytes(r.Hash))
	if err != nil {
		return nil, eth.NewErrorf(eth.EcServer, "Server error", "failed to marshal tx hash: %v", err)
	}
	return result, nil
}

// Wraps a raw Ethereum tx in a Loom SignedTx
func (t *SendRawTransactionPRCFunc) ethereumToTendermintTx(txBytes []byte) (types.Tx, error) {
	msg := &vm.MessageTx{}
	msg.Data = txBytes
	var tx etypes.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		return nil, err
	}

	if tx.To() != nil {
		msg.To = loom.Address{
			ChainID: t.chainID,
			Local:   tx.To().Bytes(),
		}.MarshalPB()
	}

	ethFrom, err := etypes.Sender(t.ethSigner, &tx)
	if err != nil {
		return nil, err
	}
	msg.From = loom.Address{
		ChainID: "eth",
		Local:   ethFrom.Bytes(),
	}.MarshalPB()

	txTx := &ltypes.Transaction{
		Id: uint32(ltypes.TxID_ETHEREUM),
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
