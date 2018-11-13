package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
)

// ValidatorsManager implements loomchain.ValidatorsManager interface
type ValidatorsManager struct {
	ctx contract.Context
}

func NewValidatorsManager(pvm *PluginVM) (*ValidatorsManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("dposV2")
	if err != nil {
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.createContractContext(caller, contractAddr, readOnly))
	return &ValidatorsManager{
		ctx: ctx,
	}, nil
}

func NewNoopValidatorsManager() *ValidatorsManager {
	var manager *ValidatorsManager
	return manager
}

func (m *ValidatorsManager) Slash(validatorAddr loom.Address) {
	if m == nil {
		return
	}
	dposv2.Slash(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) Reward(validatorAddr loom.Address) {
	if m == nil {
		return
	}
	dposv2.Reward(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) Elect() {
	// May be called with a nil receiver when DPOSv2 contract is not deployed
	if m == nil {
		return
	}
	dposv2.Elect(m.ctx)
}
