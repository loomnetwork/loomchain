package rpc

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/go-loom/auth"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	lauth "github.com/loomnetwork/loomchain/auth"
)

const (
	deployId = uint32(1)
	callId   = uint32(2)
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
	ethTx, from, err := tendermintToEthereumTx(tx)
	if err != nil {
		return nil, err
	}
	mt.txs = append([]EthTxInfo{{ethTx, from}}, mt.txs...)
	return &ctypes.ResultBroadcastTx{}, nil
}

func tendermintToEthereumTx(tmTx types.Tx) (*etypes.Transaction, common.Address, error) {
	var err error
	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(tmTx), &signedTx); err != nil {
		return nil, common.Address{}, err
	}
	local, err := lauth.VerifySolidity66Byte(signedTx)
	if err != nil {
		return nil, common.Address{}, err
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return nil, common.Address{}, err
	}

	var txTx ltypes.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return nil, common.Address{}, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return nil, common.Address{}, err
	}

	var ethTx *etypes.Transaction
	switch txTx.Id {
	case deployId:
		{
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return nil, common.Address{}, err
			}
			ethTx = etypes.NewContractCreation(
				nonceTx.Sequence,
				big.NewInt(deployTx.Value.Value.Int64()),
				0,
				big.NewInt(0),
				deployTx.Code,
			)

		}
	case callId:
		{
			var callTx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &callTx); err != nil {
				return nil, common.Address{}, err
			}
			ethTx = etypes.NewTransaction(
				nonceTx.Sequence,
				common.BytesToAddress(msg.To.Local),
				big.NewInt(callTx.Value.Value.Int64()),
				0,
				big.NewInt(0),
				callTx.Input,
			)
		}
	default:
		return nil, common.Address{}, fmt.Errorf("unrecognised ethereum tx index %v", txTx.Id)
	}

	return ethTx, common.BytesToAddress(local), nil
}

func DefaultChainConfig() *params.ChainConfig {
	cliqueCfg := params.CliqueConfig{
		Period: 10,   // Number of seconds between blocks to enforce
		Epoch:  1000, // Epoch length to reset votes and checkpoint
	}
	return &params.ChainConfig{
		ChainID:        big.NewInt(0), // Chain id identifies the current chain and is used for replay protection
		HomesteadBlock: nil,           // Homestead switch block (nil = no fork, 0 = already homestead)
		DAOForkBlock:   nil,           // TheDAO hard-fork switch block (nil = no fork)
		DAOForkSupport: true,          // Whether the nodes supports or opposes the DAO hard-fork
		// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
		EIP150Block:         nil,                                  // EIP150 HF block (nil = no fork)
		EIP150Hash:          common.BytesToHash([]byte("myHash")), // EIP150 HF hash (needed for header only clients as only gas pricing changed)
		EIP155Block:         big.NewInt(0),                        // EIP155 HF block
		EIP158Block:         big.NewInt(0),                        // EIP158 HF block
		ByzantiumBlock:      big.NewInt(0),                        // Byzantium switch block (nil = no fork, 0 = already on byzantium)
		ConstantinopleBlock: nil,                                  // Constantinople switch block (nil = no fork, 0 = already activated)
		// Various consensus engines
		Ethash: new(params.EthashConfig),
		Clique: &cliqueCfg,
	}
}
