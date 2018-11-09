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

func (m *ValidatorsManager) Slash(validatorAddr loom.Address) {
	dposv2.Slash(m.ctx, validatorAddr)
}
