package auth

type Config struct {
	// Per-chain tx signing config, indexed by chain ID
	Chains map[string]ChainConfig
}

type ChainConfig struct {
	TxType SignedTxType
	AccountType
}

func DefaultConfig() *Config {
	chains := make(map[string]ChainConfig)
	chains["default"] = ChainConfig{TxType: "loom"}
	chains["eth"] = ChainConfig{TxType: "eth", AccountType: 1}

	return &Config{
		Chains: chains,
	}
}

// Clone returns a deep clone of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c
	clone.Chains = make(map[string]ChainConfig, len(c.Chains))
	for k, v := range c.Chains {
		clone.Chains[k] = v
	}
	return &clone
}

func (c *Config) AddressMapperContractRequired() bool {
	for _, v := range c.Chains {
		if v.AccountType == MappedAccountType {
			return true
		}
	}
	return false
}
