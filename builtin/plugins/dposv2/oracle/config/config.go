package config

type EthClientSerializableConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Path of Private key that should be used to sign txs sent to Ethereum
	PrivateKeyPath string
}

type DAppChainSerializableConfig struct {
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom DAppChain
	PrivateKeyPath string
}

type TimeLockWorkerSerializableConfig struct {
	TimeLockFactoryHexAddress string
	Enabled                   bool
}

type OracleSerializableConfig struct {
	Enabled              bool
	StatusServiceAddress string
	DAppChainCfg         DAppChainSerializableConfig
	EthClientCfg         *EthClientSerializableConfig
	TimeLockWorkerCfg    *TimeLockWorkerSerializableConfig
	MainnetPollInterval  int64
}

func DefaultConfig() *OracleSerializableConfig {
	// Default config disables oracle, so
	// no need to populate oracle config
	// with default vaule.
	return &OracleSerializableConfig{
		Enabled: false,
	}
}

// Clone returns a deep clone of the config.
func (c *EthClientSerializableConfig) Clone() *EthClientSerializableConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

// Clone returns a deep clone of the config.
func (c *DAppChainSerializableConfig) Clone() *DAppChainSerializableConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

// Clone returns a deep clone of the config.
func (c *OracleSerializableConfig) Clone() *OracleSerializableConfig {
	if c == nil {
		return nil
	}
	clone := *c
	clone.DAppChainCfg = c.DAppChainCfg.Clone()
	clone.EthClientCfg = c.EthClientCfg.Clone()
	return &clone
}
