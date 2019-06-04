package vm

import (
	"fmt"

	proto "github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	etypes "github.com/ethereum/go-ethereum/core/types"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	registry "github.com/loomnetwork/loomchain/registry/factory"
)

type EthTxHandler struct {
	*Manager
	CreateRegistry registry.RegistryFactoryFunc
}

func (h *EthTxHandler) ProcessTx(
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

	var tx EthTx
	err = proto.Unmarshal(msg.Data, &tx)
	if err != nil {
		return r, err
	}

	vm, err := h.Manager.InitVM(VMType_EVM, state)
	if err != nil {
		return r, err
	}

	ethTx := &etypes.Transaction{}
	if err := ethTx.UnmarshalJSON(tx.EthereumTransaction); err != nil {
		return r, err
	}
	value := &loom.BigUInt{ethTx.Value()}
	if ethTx.To() == nil {
		retCreate, addr, errCreate := vm.Create(origin, ethTx.Data(), value)

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
		_ = reg.Register("", addr, caller)

		r.Info = utils.DeployEvm
		return r, nil
	} else {
		to := loom.UnmarshalAddressPB(msg.To)
		r.Data, err = vm.Call(origin, to, ethTx.Data(), value)
		if err != nil {
			return r, err
		}
		r.Info = utils.CallEVM
		return r, err
	}
}

type DeployTxHandler struct {
	*Manager
	CreateRegistry registry.RegistryFactoryFunc
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

	retCreate, addr, errCreate := vm.Create(origin, tx.Code, value)

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
	reg.Register(tx.Name, addr, caller)

	if tx.VmType == VMType_EVM {
		r.Info = utils.DeployEvm
	} else {
		r.Info = utils.DeployPlugin
	}
	return r, nil
}

type CallTxHandler struct {
	*Manager
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
	r.Data, err = vm.Call(origin, addr, tx.Input, value)
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
