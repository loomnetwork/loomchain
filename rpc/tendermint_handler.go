package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

const (
	ethChainID = "eth"
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
	switch t.name {
	case "eth_sendRawTransaction":
		var err *eth.Error
		var lErr error
		ethTx, err := t.TranslateSendRawTransactionParams(input)
		if err != nil {
			return nil, err
		}
		tmTx, lErr = ethereumToTendermintTx(ethTx)
		if lErr != nil {
			return nil, eth.NewErrorf(eth.EcServer, "Parse parameters", "convert ethereum tx to tendermint tx, error %v", lErr)
		}

	default:
		return nil, eth.NewError(eth.EcParseError, "Parse parameters", "unknown method")
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
	var tx types.Transaction
	if err := tx.UnmarshalJSON(txBytes); err != nil {
		return nil, eth.NewErrorf(eth.EcParseError, "Parse params", "unmarshalling ethereum transaction, %v", err)
	}

	var err error
	msg := &vm.MessageTx{}
	txTx := &ltypes.Transaction{}
	if tx.To() == nil {
		deployTx := &vm.DeployTx{
			VmType: vm.VMType_EVM,
			Code:   txBytes,
			Name:   "",
			Value:  &ltypes.BigUInt{Value: loom.BigUInt{tx.Value()}},
		}
		msg.Data, err = proto.Marshal(deployTx)
		txTx.Id = deployId
	} else {
		callTx := &vm.CallTx{
			VmType: vm.VMType_EVM,
			Input:  txBytes,
			Value:  &ltypes.BigUInt{Value: loom.BigUInt{tx.Value()}},
		}
		msg.Data, err = proto.Marshal(callTx)
		txTx.Id = callId
	}
	msg.To = EthToLoomAddress(*tx.To()).MarshalPB()
	//fmt.Println("to", EthToLoomAddress(*tx.To()))
	//chainConfig := evm.DefaultChainConfig()
	//ethSigner := etypes.MakeSigner(&chainConfig, blockNumber)
	//sender, err :=  etypes.Sender(ethSigner, tx)
	//if err != nil {
	//	return nil, err
	//}
	//fmt.Println("sender", sender.String())
	//msg.From = EthToLoomAddress(sender).MarshalPB()
	loomKey, err := crypto.GenerateKey()
	loomSigner := &auth.EthSigner66Byte{PrivateKey: loomKey}
	msg.From = loom.Address{
		ChainID: "ethTransaction",
		Local:   loomSigner.PublicKey(),
	}.MarshalPB()
	fmt.Println("loomsinger.Publickey", loomSigner.PublicKey())
	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(loomKey.PublicKey).Hex())
	fmt.Println("loomsinger.local", local)

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
	nonceTxBytes, err := proto.Marshal(nonceTx)
	if err != nil {
		return nil, err
	}

	//loomKey, err := crypto.GenerateKey()
	//loomSigner := &auth.EthSigner66Byte{PrivateKey: loomKey}
	return proto.Marshal(auth.SignTx(loomSigner, nonceTxBytes))
}

/**/
