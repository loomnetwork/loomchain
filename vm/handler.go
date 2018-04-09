package vm

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
	tmcommon "github.com/tendermint/tmlibs/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/core/state"
)

//var vmPrefix = []byte("vm")

func ProcessSendTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	//vmState := store.PrefixKVStore(state, vmPrefix)
	//vmState.Set(tx.To.Local, tx.Code)

	cfg := getConfig()
	ethDB :=  NewEvmStore(loomState)
	cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(ethDB))

	res, _, err := runtime.Call(common.BytesToAddress(tx.To.Local), tx.Code, &cfg)

	cfg.State.Commit(true)

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	r.Tags = append(r.Tags,kvpResult)
	return r, err
}

func ProcessDeployTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	//vmState := store.PrefixKVStore(state, vmPrefix)
	//vmState.Set(tx.To.Local, tx.Code)

	cfg := getConfig()
	ethDB :=  NewEvmStore(loomState)
	cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(ethDB))

	res, addr, _, err := runtime.Create(tx.Code, &cfg)

	cfg.State.Commit(true)

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	kvpAddr := tmcommon.KVPair{[]byte{1}, addr[:]}
	r.Tags = append(r.Tags,kvpResult)
	r.Tags = append(r.Tags,kvpAddr)
	return r, err
}
