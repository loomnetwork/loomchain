package vm

import (
	"github.com/gogo/protobuf/proto"

	"loom"
	"loom/store"
)

const vmPrefix = []byte("vm")

func ProcessDeployTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	vmState := store.PrefixKVStore(state, vmPrefix)
	vmState.Set(tx.To, tx.Code)

	return r, nil
}
