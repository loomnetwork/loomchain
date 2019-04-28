package evm

type EvmStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
}

func DefaultEvmStoreConfig() *EvmStoreConfig {
	return &EvmStoreConfig{
		DBName:    "evm",
		DBBackend: "goleveldb",
	}
}

// Clone returns a deep clone of the config.
func (cfg *EvmStoreConfig) Clone() *EvmStoreConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}
