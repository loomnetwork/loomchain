package plugin

import (
	"strconv"

	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
	"github.com/loomnetwork/loomchain/features"
	regcommon "github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"
)

var (
	// ErrChainConfigContractNotFound indicates that the ChainConfig contract hasn't been deployed yet.
	ErrChainConfigContractNotFound = errors.New("[ChainConfigManager] ChainContract contract not found")
)

// ChainConfigManager implements loomchain.ChainConfigManager interface
type ChainConfigManager struct {
	ctx   contract.Context
	state loomchain.State
	build uint64
}

// NewChainConfigManager attempts to create an instance of ChainConfigManager.
func NewChainConfigManager(pvm *PluginVM, state loomchain.State) (*ChainConfigManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("chainconfig")
	if err != nil {
		if err == regcommon.ErrNotFound {
			return nil, ErrChainConfigContractNotFound
		}
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.CreateContractContext(caller, contractAddr, readOnly))
	build, err := strconv.ParseUint(loomchain.Build, 10, 64)
	if err != nil {
		build = 0
	}
	return &ChainConfigManager{
		ctx:   ctx,
		state: state,
		build: build,
	}, nil
}

// EnableFeatures activates feature flags.
func (c *ChainConfigManager) EnableFeatures(blockHeight int64) error {
	featureList, err := chainconfig.EnableFeatures(c.ctx, uint64(blockHeight), c.build)
	if err != nil {
		// When an unsupported feature has been activated by the rest of the chain
		// panic to prevent the node from processing any further blocks until it's
		// upgraded to a new build that supports the feature.
		if err == chainconfig.ErrFeatureNotSupported {
			panic(err)
		}
		return err
	}

	var minRequiredBuild uint64
	for _, feature := range featureList {
		c.state.SetFeature(feature.Name, true)
		if feature.BuildNumber > minRequiredBuild {
			minRequiredBuild = feature.BuildNumber
		}
	}

	if c.state.FeatureEnabled(features.ChainCfgVersion1_4, false) &&
		(minRequiredBuild > c.state.GetMinBuildNumber()) {
		c.state.SetMinBuildNumber(minRequiredBuild)
	}
	return nil
}

// UpdateConfig applies pending config changes to the on-chain config and returns the number of config changes
func (c *ChainConfigManager) UpdateConfig() (int, error) {
	if !c.state.FeatureEnabled(features.ChainCfgVersion1_3, false) {
		return 0, nil
	}

	settings, err := chainconfig.HarvestPendingActions(c.ctx, c.build)
	if err != nil {
		return 0, err
	}

	var minRequiredBuild uint64
	for _, setting := range settings {
		if err := c.state.ChangeConfigSetting(setting.Name, setting.Value); err != nil {
			c.ctx.Logger().Error("failed to apply config change", "key", setting.Name, "err", err)
		} else if setting.BuildNumber > minRequiredBuild {
			minRequiredBuild = setting.BuildNumber
		}
	}

	if c.state.FeatureEnabled(features.ChainCfgVersion1_4, false) &&
		(minRequiredBuild > c.state.GetMinBuildNumber()) {
		c.state.SetMinBuildNumber(minRequiredBuild)
	}
	return len(settings), nil
}
