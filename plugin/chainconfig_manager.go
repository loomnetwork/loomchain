package plugin

import (
	"strconv"

	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
	regcommon "github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"
)

var (
	// ErrChainConfigContractNotFound indicates that the ChainConfig contract hasn't been deployed yet.
	ErrChainConfigContractNotFound = errors.New("[ChainConfigManager] ChainContract contract not found")
)

const (
	// This configPrefix must have the same value as configPrefix in app.go
	configPrefix = "config"
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

func (c *ChainConfigManager) EnableFeatures(blockHeight int64) error {
	features, err := chainconfig.EnableFeatures(c.ctx, uint64(blockHeight), c.build)
	if err != nil {
		// When an unsupported feature has been activated by the rest of the chain
		// panic to prevent the node from processing any further blocks until it's
		// upgraded to a new build that supports the feature.
		if err == chainconfig.ErrFeatureNotSupported {
			panic(err)
		}
		return err
	}
	for _, feature := range features {
		c.state.SetFeature(feature.Name, true)
	}
	return nil
}

func (c *ChainConfigManager) UpdateConfig() error {
	cfgSettings, err := chainconfig.UpdateConfig(c.ctx)
	if err != nil {
		return err
	}
	for _, cfgSetting := range cfgSettings {
		if cfgSetting.Status == chainconfig.CfgSettingRemoving {
			c.state.RemoveCfgSetting(cfgSetting.Name)
		} else if cfgSetting.Status == chainconfig.CfgSettingActivated {
			c.state.SetCfgSetting(cfgSetting)
		}
	}
	return nil
}
