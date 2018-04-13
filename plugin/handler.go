package plugin

import (
	"errors"

	proto "github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loom"
)

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

	var entry ContractEntry
	err = proto.Unmarshal(msg.Data, &entry)
	if err != nil {
		return r, err
	}

	if entry.VmType != VMType_PLUGIN {
		return r, errors.New("only plugin VM supported")
	}

	vm := &PluginVM{
		Loader: h.Loader,
		State:  state,
	}

	_, err = vm.Call(caller, addr, entry.Code)
	return r, err
}
