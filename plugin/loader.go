package plugin

import (
	"errors"

	"github.com/loomnetwork/go-loom/plugin"
)

var (
	ErrPluginNotFound = errors.New("plugin not found")
)

type Loader interface {
	LoadContract(name string) (plugin.Contract, error)
}

type MultiLoader struct {
	loaders []Loader
}

func NewMultiLoader(loaders ...Loader) *MultiLoader {
	return &MultiLoader{
		loaders: loaders,
	}
}

func (m *MultiLoader) LoadContract(name string) (plugin.Contract, error) {
	if len(m.loaders) == 0 {
		return nil, errors.New("no loaders specified")
	}

	for _, loader := range m.loaders {
		contract, err := loader.LoadContract(name)
		if err == ErrPluginNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		return contract, nil
	}

	return nil, ErrPluginNotFound
}
