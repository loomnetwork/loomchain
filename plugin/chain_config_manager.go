package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
)

// ChainConfigManager implements loomchain.ChainConfigManager interface
type ChainConfigManager struct {
	ctx contract.Context
}

func NewChainConfigManager(pvm *PluginVM) (*ChainConfigManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("chainconfig")
	if err != nil {
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.createContractContext(caller, contractAddr, readOnly))
	return &ChainConfigManager{
		ctx: ctx,
	}, nil
}

func (c *ChainConfigManager) CheckAndEnablePendingFeatures() ([]*chainconfig.FeatureInfo, error) {
	featureInfos, err := chainconfig.FeatureList(c.ctx)
	if err != nil {
		return nil, err
	}
	for _, featureInfo := range featureInfos {
		if featureInfo.Feature.Status == chainconfig.FeaturePending && featureInfo.Percentage > 66 {
			featureInfo.Feature.Status = chainconfig.FeatureEnabled
			chainconfig.UpdateFeature(c.ctx, featureInfo.Feature)
		}
	}
	return featureInfos, nil
}
