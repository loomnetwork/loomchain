package plugin

import (
	"errors"

	proto "github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loom"
)

type DeployTxHandler struct {
	Loader
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

	var caller, addr loom.Address
	caller.UnmarshalPB(msg.From)
	addr.UnmarshalPB(msg.To)

	var tx DeployTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	if tx.VmType != VMType_PLUGIN {
		return r, errors.New("only plugin VM supported")
	}

	vm := &PluginVM{
		Loader: h.Loader,
		State:  state,
	}

	_, _, err = vm.Create(caller, tx.Code)
	return r, err
}

type CallTxHandler struct {
	Loader
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

	var caller, addr loom.Address
	caller.UnmarshalPB(msg.From)
	addr.UnmarshalPB(msg.To)

	var tx CallTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	vm := &PluginVM{
		Loader: h.Loader,
		State:  state,
	}

	_, err = vm.Call(caller, addr, tx.Input)
	return r, err
}
