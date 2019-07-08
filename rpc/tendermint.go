package rpc

import (
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom/auth"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"

	lauth "github.com/loomnetwork/loomchain/auth"
)

type TendermintRpc interface {
	BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error)
}

type RuntimeTendermintRpc struct{}

func (t RuntimeTendermintRpc) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	return core.BroadcastTxSync(tx)
}

type EthTxInfo struct {
	Tx   *etypes.Transaction
	From common.Address
}

type MockTendermintRpc struct {
	txs []EthTxInfo
}

func (mt *MockTendermintRpc) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	ethTx, err := tendermintToEthereumTx(tx)
	if err != nil {
		return nil, err
	}

	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(tx), &signedTx); err != nil {
		return nil, err
	}
	from, err := lauth.VerifySolidity66Byte(signedTx)
	if err != nil {
		return nil, err
	}

	mt.txs = append([]EthTxInfo{{ethTx, common.BytesToAddress(from)}}, mt.txs...)
	return &ctypes.ResultBroadcastTx{}, nil
}

func tendermintToEthereumTx(tmTx types.Tx) (*etypes.Transaction, error) {
	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(tmTx), &signedTx); err != nil {
		return nil, err
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return nil, err
	}

	var txTx ltypes.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return nil, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return nil, err
	}

	var tx etypes.Transaction
	if err := rlp.DecodeBytes(msg.Data, &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}
