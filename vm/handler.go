package vm

import (
	proto "github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loom"
)

type DeployTxHandler struct {
	*Manager
}

func (h *DeployTxHandler) ProcessTx(
	state loom.State,
	txBytes []byte,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	var msg MessageTx
	err := proto.Unmarshal(txBytes, &msg)
	if err != nil {
		return r, err
	}

	caller := loom.UnmarshalAddressPB(msg.From)

	var tx DeployTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	vm, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	_, _, err = vm.Create(caller, tx.Code)
	return r, err
}

type CallTxHandler struct {
	*Manager
}

func (h *CallTxHandler) ProcessTx(
	state loom.State,
	txBytes []byte,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	var msg MessageTx
	err := proto.Unmarshal(txBytes, &msg)
	if err != nil {
		return r, err
	}

	caller := loom.UnmarshalAddressPB(msg.From)
	addr := loom.UnmarshalAddressPB(msg.To)

	var tx CallTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	vm, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	_, err = vm.Call(caller, addr, tx.Input)
	return r, err
}
