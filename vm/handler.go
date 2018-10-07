package vm

import (
	"fmt"

	proto "github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	registry "github.com/loomnetwork/loomchain/registry"
	regFactory "github.com/loomnetwork/loomchain/registry/factory"
)

type DeployTxHandler struct {
	*Manager
	CreateRegistry       regFactory.RegistryFactoryFunc
	GenesisContractOwner loom.Address
}

func getInitialVersionOfContract(reg registry.Registry, contractAddr loom.Address) (string, error) {
	record, err := reg.GetRecord(contractAddr)
	if err != nil {
		if err != registry.ErrNotImplemented {
			return registry.DefaultContractVersion, err
		}
		return registry.DefaultContractVersion, nil
	}

	return record.InitialVersion, nil
}

func validateInitAttempt(
	reg registry.Registry,
	caller loom.Address,
	genesisContractOwner loom.Address,
	contractName,
	contractVersion string) error {

	// Try to resolve, if we found it, that means contract with
	// this version already exists, and if it is any other error than
	// not found, we should return that error.
	addr, err := reg.Resolve(contractName, contractVersion)

	if contractVersion == registry.DefaultContractVersion {
		// In previous build, we were not checking, if contract is already registered.
		// So to maintain backward compatibility we are skipping that check here.
		return nil
	}

	if err == nil {
		return fmt.Errorf("contract with name: %s and version: %s already exists", contractName, contractVersion)
	}
	if err != registry.ErrNotFound {
		return err
	}

	// Get master entry. If it doesnt exists, than
	// it means plugin is being registered for first time
	// otherwise proceed with validation.
	addr, err = reg.Resolve(contractName, registry.DefaultContractVersion)
	if err != nil {
		if err == registry.ErrNotFound {
			return nil
		}
		return err
	}

	// If control flow reaches here, than it must be registry version 2 or greater
	record, err := reg.GetRecord(addr)
	if err != nil {
		return err
	}

	if addr.Compare(loom.UnmarshalAddressPB(record.Owner)) == 0 {
		if caller.Compare(genesisContractOwner) != 0 {
			return fmt.Errorf("owner of initial version doesnt match caller.")
		}
	} else {
		if caller.Compare(loom.UnmarshalAddressPB(record.Owner)) != 0 {
			return fmt.Errorf("owner of initial version doesnt match caller.")
		}
	}

	return nil
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

	reg := h.CreateRegistry(state)
	err = validateInitAttempt(reg, caller, h.GenesisContractOwner, tx.Name, tx.ContractVersion)
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

	retCreate, addr, errCreate := vm.Create(origin, tx.ContractVersion, tx.Code, value)

	response, errMarshal := proto.Marshal(&DeployResponse{
		Contract: &types.Address{
			ChainId: addr.ChainID,
			Local:   addr.Local,
		},
		Output: retCreate,
	})
	if errMarshal != nil {
		if errCreate != nil {
			return r, errors.Wrapf(errCreate, "[DeployTxHandler] Error deploying EVM contract on create")
		} else {
			return r, errors.Wrapf(errMarshal, "[DeployTxHandler] Error deploying EVM contract on marshaling evm error")
		}
	}
	r.Data = append(r.Data, response...)
	if errCreate != nil {
		return r, errors.Wrapf(errCreate, "[DeployTxHandler] Error deploying EVM contract on create")
	}

	if err := reg.Register(tx.Name, tx.ContractVersion, addr, caller); err != nil {
		// Earlier build were sloppy and didn't check for an error, so to maintain backwards compatibility
		// ignore the error if a specific contract version isn't provided in the tx.
		if tx.ContractVersion != registry.DefaultContractVersion {
			return r, err
		}
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
	CreateRegistry regFactory.RegistryFactoryFunc
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

	if tx.ContractVersion == registry.DefaultContractVersion {
		reg := h.CreateRegistry(state)
		initialVersion, err := getInitialVersionOfContract(reg, addr)
		if err != nil {
			return r, err
		}
		tx.ContractVersion = initialVersion
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
	r.Data, err = vm.Call(origin, addr, tx.ContractVersion, tx.Input, value)
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
