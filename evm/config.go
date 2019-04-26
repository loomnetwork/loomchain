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
func (EvmStore *EvmStoreConfig) Clone() *EvmStoreConfig {
	if EvmStore == nil {
		return nil
	}
	clone := *EvmStore
	return &clone
}
