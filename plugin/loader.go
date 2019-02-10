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
	loaders []Loader
}

func NewMultiLoader(loaders ...Loader) *MultiLoader {
	return &MultiLoader{
		loaders: loaders,
	}
}

func (m *MultiLoader) LoadContract(name string, blockHeight int64) (plugin.Contract, error) {
	if len(m.loaders) == 0 {
		return nil, errors.New("no loaders specified")
	}

	for _, loader := range m.loaders {
		contract, err := loader.LoadContract(name, blockHeight)
		if err == ErrPluginNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

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
// onwards. An override can itself be overriden by another override with a higher block height.
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
	if overrides, _ := m.overrides[name]; overrides != nil {
		for i := len(overrides) - 1; i >= 0; i-- {
			if blockHeight >= overrides[i].BlockHeight {
				return overrides[i].Contract
			}
		}
	}
	return nil
}
