package evm

type EvmStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
	// CacheSizeMegs defines cache size (in megabytes) of EVM store
	CacheSizeMegs int
}

func DefaultEvmStoreConfig() *EvmStoreConfig {
	return &EvmStoreConfig{
		DBName:        "evm",
		DBBackend:     "goleveldb",
		CacheSizeMegs: 20,
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
