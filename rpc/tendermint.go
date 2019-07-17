package rpc

import (
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain/evm/utils"
)

const (
	ethTxId = 4
)

type TendermintRpc interface {
	BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error)
	ethereumToTendermintTx(txBytes []byte) (types.Tx, error)
}

type RuntimeTendermintRpc struct{
	LoomServer
}

func (t RuntimeTendermintRpc) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	return core.BroadcastTxSync(tx)
}

func (t RuntimeTendermintRpc) ethereumToTendermintTx(txBytes []byte) (types.Tx, error) {
	msg := &vm.MessageTx{}
	msg.Data = txBytes

	var tx etypes.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		return nil, err
	}

	if tx.To() != nil {
		msg.To = loom.Address{
			ChainID: t.ChainID,
			Local:   tx.To().Bytes(),
		}.MarshalPB()
	}

	chainConfig := utils.DefaultChainConfig()
	ethSigner := etypes.MakeSigner(&chainConfig, blockNumber)
	ethFrom, err := etypes.Sender(ethSigner, &tx)
	if err != nil {
		return nil, err
	}
	from, err := t.localToEthAccount(ethFrom.Bytes())
	if err != nil {
		return nil, err
	}
	msg.From = from.MarshalPB()

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
