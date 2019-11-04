package plugin

import (
	"errors"
	"github.com/loomnetwork/go-loom/plugin"
)

var (
	ErrPluginNotFound = errors.New("plugin not found")
)

type Loader interface {
	LoadContract(name string, blockHeight int64) (plugin.Contract, error)
	UnloadContracts()
}

type MultiLoader struct {
	loaders                []Loader
	knownSuccessfulLoaders map[string]Loader
}

func NewMultiLoader(loaders ...Loader) *MultiLoader {
	return &MultiLoader{
		loaders:                loaders,
		knownSuccessfulLoaders: map[string]Loader{},
	}
}

func (m *MultiLoader) LoadContract(name string, blockHeight int64) (plugin.Contract, error) {
	if len(m.loaders) == 0 {
		return nil, errors.New("no loaders specified")
	}

	// The assumption is that once a specific loader has successfully loaded a plugin
	// by a name once, it'll be able to load it the next time as well.
	// This saves resources on almost never having to try loading with the whole loader list
	knownSuccessfulLoader := m.knownSuccessfulLoaders[name]
	if knownSuccessfulLoader != nil {
		contract, err := knownSuccessfulLoader.LoadContract(name, blockHeight)
		if err == nil {
			return contract, nil
		} else if err != ErrPluginNotFound {
			return nil, err
		}
	}

	for _, loader := range m.loaders {
		// An attempt to load with a known loader was already made and failed, no point retrying
		if knownSuccessfulLoader != nil && loader == knownSuccessfulLoader {
			continue
		}

		contract, err := loader.LoadContract(name, blockHeight)
		if err == ErrPluginNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		m.knownSuccessfulLoaders[name] = loader
		return contract, nil
	}

	return nil, ErrPluginNotFound
}

func (m *MultiLoader) UnloadContracts() {
	for _, loader := range m.loaders {
		loader.UnloadContracts()
	}
}

// ContractOverride specifies a contract that should be loaded instead of another contract.
// The override kicks in at a particular block height, and remains in force from that height
// onwards. An override can itself be overridden by another override with a higher block height.
type ContractOverride struct {
	plugin.Contract
	// Height at which the override should take effect
	BlockHeight int64
}

// ContractOverrideMap maps a contract name:version to override info
type ContractOverrideMap = map[string][]*ContractOverride

type StaticLoader struct {
	Contracts []plugin.Contract
	overrides ContractOverrideMap
}

func NewStaticLoader(contracts ...plugin.Contract) *StaticLoader {
	return &StaticLoader{
		Contracts: contracts,
	}
}

func (m *StaticLoader) SetContractOverrides(overrides ContractOverrideMap) {
	m.overrides = overrides
}

func (m *StaticLoader) UnloadContracts() {}

func (m *StaticLoader) LoadContract(name string, blockHeight int64) (plugin.Contract, error) {
	meta, err := ParseMeta(name)
	if err != nil {
		return nil, err
	}
	if contract := m.findOverride(name, blockHeight); contract != nil {
		return contract, nil
	}
	for _, contract := range m.Contracts {
		contractMeta, err := contract.Meta()
		if err != nil {
			return nil, err
		}
		if compareMeta(meta, &contractMeta) == 0 {
			return contract, nil
		}
	}

	return nil, ErrPluginNotFound
}

func (m *StaticLoader) findOverride(name string, blockHeight int64) plugin.Contract {
	if overrides := m.overrides[name]; overrides != nil {
		for i := len(overrides) - 1; i >= 0; i-- {
			if blockHeight >= overrides[i].BlockHeight {
				return overrides[i].Contract
			}
		}
	}
	return nil
}
