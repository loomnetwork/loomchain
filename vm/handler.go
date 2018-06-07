package vm

import (
	proto "github.com/gogo/protobuf/proto"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry"
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

	retCreate, addr, errCreate := vm.Create(caller, tx.Code)

	response, errMarshal := proto.Marshal(&DeployResponse{
		Contract: &types.Address{
			ChainId: addr.ChainID,
			Local:   addr.Local,
		},
		Output: retCreate,
	})
	if errMarshal != nil {
		return r, errMarshal
	}
	r.Data = append(r.Data, response...)
	if errCreate != nil {
		return r, errCreate
	}

	if len(tx.Name) > 0 {
		reg := &registry.StateRegistry{
			State: state,
		}
		reg.Register(tx.Name, addr, caller)
	}
	return r, nil
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
