package config

type EthClientSerializableConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Plasma contract address on Ethereum
	PlasmaHexAddress string
	// Path of Private key that should be used to sign txs sent to Ethereum
	PrivateKeyPath string
	// Override default gas computation when sending txs to Ethereum
	OverrideGas bool
	// How often Ethereum should be polled for mined txs
	TxPollInterval int64
	// Maximum amount of time to way for a tx to be mined by Ethereum
	TxTimeout int64
}

type DAppChainSerializableConfig struct {
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom DAppChain
	PrivateKeyPath string
	// Plasma cash contract on d app chain
	ContractName string
}

type OracleSerializableConfig struct {
	PlasmaBlockInterval  uint32
	StatusServiceAddress string
	DAppChainCfg         *DAppChainSerializableConfig
	EthClientCfg         *EthClientSerializableConfig
}

type PlasmaCashSerializableConfig struct {
	OracleEnabled   bool
	ContractEnabled bool

	OracleConfig *OracleSerializableConfig
}

func DefaultConfig() *PlasmaCashSerializableConfig {
	// Default config disables oracle, so
	// no need to populate oracle config
	// with default vaule.
	return &PlasmaCashSerializableConfig{
		OracleEnabled:   false,
		ContractEnabled: false,
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

// Clone returns a deep clone of the config.
func (c *PlasmaCashSerializableConfig) Clone() *PlasmaCashSerializableConfig {
	if c == nil {
		return nil
	}
	clone := *c
	clone.OracleConfig = c.OracleConfig.Clone()
	return &clone
}
