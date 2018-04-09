package vm

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/state"
	tmcommon "github.com/tendermint/tmlibs/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

type evmState struct {
	loom.State
	evmDB state.StateDB
}

var vmPrefix = []byte("vm")

func ProcessSendTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	vmState := store.PrefixKVStore(state, vmPrefix)
	vmState.Set(tx.To.Local, tx.Code)

	//Send create transaction to EVM
	//db := state.(evmState).evmDB
	//res, _, txErr := Call(common.BytesToAddress(tx.To.Local), tx.Code, &db)

	//{
	cfg := getConfig()
	sdb := state.(evmState).evmDB
	cfg.State = &sdb
	res, _, txErr := runtime.Call(common.BytesToAddress(tx.To.Local), tx.Code, &cfg)
	res = res
	txErr = txErr
	//}

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	r.Tags = append(r.Tags,kvpResult)
	return r, txErr
}

func ProcessDeployTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	vmState := store.PrefixKVStore(state, vmPrefix)
	vmState.Set(tx.To.Local, tx.Code)

	//Send create transaction to EVM
	db := state.(evmState).evmDB
	res, addr, _, txErr := Create(tx.Code, &db)

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	kvpAddr := tmcommon.KVPair{[]byte{1}, addr[:]}
	r.Tags = append(r.Tags,kvpResult)
	r.Tags = append(r.Tags,kvpAddr)
	return r, txErr
}
