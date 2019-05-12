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

func (c *ChainConfigManager) SetConfigs(blockHeight int64) error {
	configs, err := chainconfig.SetConfigs(c.ctx, uint64(blockHeight), c.build)
	if err != nil {
		// When an unsupported config has been activated by the rest of the chain
		// panic to prevent the node from processing any further blocks until it's
		// upgraded to a new build that supports the config.
		if err == chainconfig.ErrConfigNotSupported {
			panic(err)
		}
		return err
	}
	for _, config := range configs {
		c.state.SetConfig(config.Name, config.Settlement.Value)
	}

	// The following logic remove configs that do not
	configListHashMap := make(map[string]bool)
	configListOnChain := c.state.Range([]byte(configPrefix))
	configListOnContract, err := chainconfig.ConfigList(c.ctx)
	if err != nil {
		return err
	}

	// Make hashmap of configs on chain
	for _, config := range configListOnChain {
		configListHashMap[string(config.Key)] = true
	}

	// Cross out configs that still exist on the contract
	for _, config := range configListOnContract {
		configListHashMap[string(config.Name)] = false
	}

	// Delete configs (on chain) that do not exist anymore
	for configName, deleted := range configListHashMap {
		if deleted {
			c.state.DeleteConfig(configName)
		}
	}

	return nil
}
