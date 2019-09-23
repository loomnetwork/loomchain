package config

import (
	"github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
)

type GoContractDeployerWhitelistConfig struct {
	Enabled             bool
	DeployerAddressList []string
}

func DefaultGoContractDeployerWhitelistConfig() *GoContractDeployerWhitelistConfig {
	return &GoContractDeployerWhitelistConfig{
		Enabled: false,
	}
}

func (c *GoContractDeployerWhitelistConfig) DeployerAddresses(chainId string) ([]loom.Address, error) {
	deployerAddressList := make([]loom.Address, 0, len(c.DeployerAddressList))
	for _, addrStr := range c.DeployerAddressList {
		addr, err := loom.ParseAddress(addrStr)
		if err != nil {
			addr, err = loom.ParseAddress(chainId + ":" + addrStr)
			if err != nil {
				return nil, errors.Wrapf(err, "parsing deploy address %s", addrStr)
			}
		}
		deployerAddressList = append(deployerAddressList, addr)
	}
	return deployerAddressList, nil
}

type TxLimiterConfig struct {
	// Enables the tx limiter middleware
	Enabled bool
	// Number of seconds each session lasts
	SessionDuration int64
	// Maximum number of txs that should be allowed per session
	MaxTxsPerSession int64
}

func DefaultTxLimiterConfig() *TxLimiterConfig {
	return &TxLimiterConfig{
		SessionDuration:  60,
		MaxTxsPerSession: 60,
	}
}

// Clone returns a deep clone of the config.
func (c *TxLimiterConfig) Clone() *TxLimiterConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type ContractTxLimiterConfig struct {
	// Enables the middleware
	Enabled bool
	// Number of seconds each refresh lasts
	ContractDataRefreshInterval int64
	TierDataRefreshInterval     int64
}

func DefaultContractTxLimiterConfig() *ContractTxLimiterConfig {
	return &ContractTxLimiterConfig{
		Enabled:                     false,
		ContractDataRefreshInterval: 15 * 60,
		TierDataRefreshInterval:     15 * 60,
	}
}

// Clone returns a deep clone of the config.
func (c *ContractTxLimiterConfig) Clone() *ContractTxLimiterConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}
