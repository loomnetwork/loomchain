package auth

type Config struct {
	// Per-chain tx signing config, indexed by chain ID
	Chains map[string]ChainConfig

	//Force enable MultiChainSignatureTxMiddleware, do not enable in production!
	DebugMultiChainSignatureTxMiddlewareEnabled bool
}

type ChainConfig struct {
	TxType SignedTxType
	AccountType
}

func DefaultConfig() *Config {
	return &Config{
		Chains: make(map[string]ChainConfig),
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
