package factory

import (
	common "github.com/loomnetwork/loomchain/registry"
	registry_v1 "github.com/loomnetwork/loomchain/registry/v1"
	registry_v2 "github.com/loomnetwork/loomchain/registry/v2"
	"github.com/loomnetwork/loomchain/state"
)

type RegistryVersion int32

const (
	RegistryV1            RegistryVersion = 1
	RegistryV2            RegistryVersion = 2
	LatestRegistryVersion RegistryVersion = RegistryV2
)

// RegistryVersionFromInt safely converts an int to RegistryVersion.
func RegistryVersionFromInt(v int32) (RegistryVersion, error) {
	if v < 0 || v > int32(LatestRegistryVersion) {
		return 0, common.ErrInvalidVersion
	}
	if v == 0 {
		return RegistryV1, nil
	}
	return RegistryVersion(v), nil
}

type RegistryFactoryFunc func(state.State) common.Registry

// NewRegistryFactory returns a factory function that can be used to create a Registry instance
// matching the specified version.
func NewRegistryFactory(v RegistryVersion) (RegistryFactoryFunc, error) {
	switch v {
	case RegistryV1:
		return func(s state.State) common.Registry {
			return &registry_v1.StateRegistry{State: s}
		}, nil
	case RegistryV2:
		return func(s state.State) common.Registry {
			return &registry_v2.StateRegistry{State: s}
		}, nil
	}
	return nil, common.ErrInvalidVersion
}
