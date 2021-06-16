package vm

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/features"
	registry "github.com/loomnetwork/loomchain/registry/factory"
)

type DeployTxHandler struct {
	*Manager
	CreateRegistry         registry.RegistryFactoryFunc
	AllowNamedEVMContracts bool
	GasConsumer            func() loomchain.GasConsumer
}

func (h *DeployTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var msg MessageTx
	err := proto.Unmarshal(txBytes, &msg)
	if err != nil {
		return r, err
	}

	origin := auth.Origin(state.Context())
	caller := loom.UnmarshalAddressPB(msg.From)

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
	}

	var tx DeployTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	version1_1 := state.FeatureEnabled(features.DeployTxVersion1_1Feature, false)
	if version1_1 && (tx.VmType == VMType_EVM) && (len(tx.Name) > 0) && !h.AllowNamedEVMContracts {
		return r, errors.New("named evm contracts are not allowed")
	}

	vm, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	var value *loom.BigUInt
	if tx.Value == nil {
		value = loom.NewBigUIntFromInt(0)
	} else {
		value = &tx.Value.Value
	}

	gasConsumer := h.GasConsumer()
	retCreate, addr, errCreate := vm.Create(origin, tx.Code, value, gasConsumer)

	response, errMarshal := proto.Marshal(&DeployResponse{
		Contract: &types.Address{
			ChainId: addr.ChainID,
			Local:   addr.Local,
		},
		Output: retCreate,
	})
	if errMarshal != nil {
		if errCreate != nil {
			return r, errors.Wrapf(errCreate, "[DeployTxHandler] Error deploying contract on create")
		} else {
			return r, errors.Wrapf(errMarshal, "[DeployTxHandler] Error deploying contract on marshaling error")
		}
	}
	r.Data = append(r.Data, response...)
	if errCreate != nil {
		return r, errors.Wrapf(errCreate, "[DeployTxHandler] Error deploying contract on create")
	}

	reg := h.CreateRegistry(state)
	if err := reg.Register(tx.Name, addr, caller); err != nil && version1_1 {
		return r, err
	}

	if tx.VmType == VMType_EVM {
		r.Info = utils.DeployEvm
	} else {
		r.Info = utils.DeployPlugin
	}
	return r, nil
}

type CallTxHandler struct {
	*Manager
	GasConsumer func() loomchain.GasConsumer
}

func (h *CallTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,

) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var msg MessageTx
	err := proto.Unmarshal(txBytes, &msg)
	if err != nil {
		return r, err
	}

	origin := auth.Origin(state.Context())
	caller := loom.UnmarshalAddressPB(msg.From)
	addr := loom.UnmarshalAddressPB(msg.To)

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: %v != %v", origin, caller)
	}

	var tx CallTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	vm, err := h.Manager.InitVM(tx.VmType, state)
	if err != nil {
		return r, err
	}

	var value *loom.BigUInt
	if tx.Value == nil {
		value = loom.NewBigUIntFromInt(0)
	} else {
		value = &tx.Value.Value
	}

	gasConsumer := h.GasConsumer()
	r.Data, err = vm.Call(origin, addr, tx.Input, value, gasConsumer)
	if err != nil {
		return r, err
	}
	if tx.VmType == VMType_EVM {
		r.Info = utils.CallEVM
	} else {
		r.Info = utils.CallPlugin
	}
	return r, err
}
