package vm

import (
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
)

type DeployTxHandler struct {
	*Manager
}

func (h *DeployTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

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

	runcode, addr, err := vm.Create(caller, tx.Code)

	response, err := proto.Marshal(&DeployResponse{
		Contract: &types.Address{
			ChainId: addr.ChainID,
			Local:   addr.Local,
		},
		Output: runcode,
	})
	if err != nil {
		return r, err
	}
	r.Data = append(r.Data, response...)

	return r, err
}

type CallTxHandler struct {
	*Manager
}

func (h *CallTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

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
	if tx.GetVmType() == vm.VMType_PLUGIN {
		r.Info = "Plugin"
	} else {
		r.Info = "EVM"
	}

	vm, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	r.Data, err = vm.Call(caller, addr, tx.Input)
	if err != nil {
		return r, err
	}

	return r, err
}
