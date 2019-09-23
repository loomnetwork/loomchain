package tx_handler

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

// DeployTxHandler handles txs that deploy Go & EVM contracts
type DeployTxHandler struct {
	*vm.Manager
	CreateRegistry         registry.RegistryFactoryFunc
	AllowNamedEVMContracts bool
}

func (h *DeployTxHandler) ProcessTx(
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

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
	}

	// TODO: move the marshalling & validation above this line into middleware
	var tx vm.DeployTx
	if err := proto.Unmarshal(msg.Data, &tx); err != nil {
		return r, err
	}

	switch tx.VmType {
	case vm.VMType_EVM:
		r.Info = utils.DeployEvm

		if (len(tx.Name) > 0) && !h.AllowNamedEVMContracts {
			return r, errors.New("named evm contracts are not allowed")
		}

		// Only do basic validation of EVM deploys in CheckTx, don't execute the actual deploy
		if isCheckTx {
			return r, nil
		}

	case vm.VMType_PLUGIN:
		r.Info = utils.DeployPlugin

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

	retCreate, addr, err := vmInstance.Create(origin, tx.Code, value)
	if err != nil {
		return r, errors.Wrapf(err, "failed to create contract")
	}

	response, err := proto.Marshal(&vm.DeployResponse{
		Contract: &types.Address{
			ChainId: addr.ChainID,
			Local:   addr.Local,
		},
		Output: retCreate,
	})
	if err != nil {
		return r, errors.Wrapf(err, "failed to marshal deploy response")
	}
	r.Data = response

	reg := h.CreateRegistry(state)
	if err := reg.Register(tx.Name, addr, caller); err != nil {
		return r, err
	}
	return r, nil
}
