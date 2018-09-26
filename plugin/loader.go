package plugin

import (
	"errors"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/registry"
)

var (
	ErrPluginNotFound = errors.New("plugin not found")
)

type Loader interface {
	LoadContract(name, version string) (plugin.Contract, error)
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

func (m *MultiLoader) LoadContract(name, version string) (plugin.Contract, error) {
	if len(m.loaders) == 0 {
		return nil, errors.New("no loaders specified")
	}

	for _, loader := range m.loaders {
		contract, err := loader.LoadContract(name, version)
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

type StaticLoader struct {
	Contracts []plugin.Contract
}

func NewStaticLoader(contracts ...plugin.Contract) *StaticLoader {
	return &StaticLoader{
		Contracts: contracts,
	}
}

func (m *StaticLoader) UnloadContracts() {}

func (m *StaticLoader) LoadContract(name, version string) (plugin.Contract, error) {
	var meta *plugin.Meta
	var err error

	if version != registry.DefaultContractVersion {
		meta = &plugin.Meta{
			Name:    name,
			Version: version,
		}
	} else {
		meta, err = ParseMeta(name)
		if err != nil {
			return nil, err
		}
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
