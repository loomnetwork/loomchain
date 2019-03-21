package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
)

// ChainConfigManager implements loomchain.ChainConfigManager interface
type ChainConfigManager struct {
	ctx   contract.Context
	state loomchain.State
}

func NewChainConfigManager(pvm *PluginVM, state loomchain.State) (*ChainConfigManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("chainconfig")
	if err != nil {
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.createContractContext(caller, contractAddr, readOnly))
	return &ChainConfigManager{
		ctx:   ctx,
		state: state,
	}, nil
}

func (c *ChainConfigManager) EnableFeatures(blockHeight int64) error {
	features, err := chainconfig.EnableFeatures(c.ctx, uint64(blockHeight))
	if err != nil {
		return err
	}
	for _, feature := range features {
		c.state.SetFeature(feature.Name, true)
	}
	return nil
}
