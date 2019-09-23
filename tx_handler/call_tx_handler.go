package tx_handler

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

// CallTxHandler handles txs that call Go & EVM contracts
type CallTxHandler struct {
	*vm.Manager
}

func (h *CallTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,

) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var msg vm.MessageTx
	if err := proto.Unmarshal(txBytes, &msg); err != nil {
		return r, err
	}

	origin := auth.Origin(state.Context())
	caller := loom.UnmarshalAddressPB(msg.From)
	addr := loom.UnmarshalAddressPB(msg.To)

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: %v != %v", origin, caller)
	}

	// TODO: move the marshalling & validation above this line into middleware
	var tx vm.CallTx
	if err := proto.Unmarshal(msg.Data, &tx); err != nil {
		return r, err
	}

	switch tx.VmType {
	case vm.VMType_EVM:
		r.Info = utils.CallEVM
		// Only do basic validation of EVM calls in CheckTx, don't execute the actual call
		if isCheckTx {
			return r, nil
		}

	case vm.VMType_PLUGIN:
		r.Info = utils.CallPlugin

	default:
		return r, errors.New("invalid vm type")
	}

	vmInstance, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	var value *loom.BigUInt
	if tx.Value == nil {
		value = loom.NewBigUIntFromInt(0)
	} else {
		value = &tx.Value.Value
	}

	r.Data, err = vmInstance.Call(origin, addr, tx.Input, value)
	return r, err
}
