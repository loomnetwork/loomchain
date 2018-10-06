// +build !evm

package config

import "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/oracle"

type PlasmaCashConfig struct {
	OracleEnabled   bool
	ContractEnabled bool
	OracleConfig    *oracle.OracleConfig
}

func LoadSerializableConfig(chainID string, serializableConfig *PlasmaCashSerializableConfig) (*PlasmaCashConfig, error) {
	return &PlasmaCashConfig{
		ContractEnabled: serializableConfig.ContractEnabled,
		// Not supported in non-evm build
		OracleEnabled: false,
		OracleConfig:  nil,
	}, nil
}
