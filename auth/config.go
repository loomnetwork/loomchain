package auth

type Config struct {
	EthChainID       string
	TronChainID      string
	AddressMapping   bool
	ExternalNetworks map[string]ExternalNetworks
}

func DefaultConfig() *Config {
	return &Config{
		ExternalNetworks: make(map[string]ExternalNetworks),
	}
}

// Clone returns a deep clone of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c
	clone.ExternalNetworks = make(map[string]ExternalNetworks, len(c.ExternalNetworks))
	for k, v := range c.ExternalNetworks {
		clone.ExternalNetworks[k] = v
	}
	return &clone
}
