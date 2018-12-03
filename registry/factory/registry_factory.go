package factory

import (
	"github.com/loomnetwork/loomchain"
	common "github.com/loomnetwork/loomchain/registry"
	registry_v1 "github.com/loomnetwork/loomchain/registry/v1"
	registry_v2 "github.com/loomnetwork/loomchain/registry/v2"
)



type RegistryFactoryFunc func(loomchain.State) common.Registry

// NewRegistryFactory returns a factory function that can be used to create a Registry instance
// matching the specified version.
func NewRegistryFactory(v common.RegistryVersion) (RegistryFactoryFunc, error) {
	switch v {
	case common.RegistryV1:
		return func(s loomchain.State) common.Registry {
			return &registry_v1.StateRegistry{State: s}
		}, nil
	case common.RegistryV2:
		return func(s loomchain.State) common.Registry {
			return &registry_v2.StateRegistry{State: s}
		}, nil
	}
	return nil, common.ErrInvalidVersion
}
