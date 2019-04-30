package evm

type EvmDBConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
	// CacheSizeMegs defines cache size (in megabytes) of EVM store
	CacheSizeMegs int
}

func DefaultEvmDBConfig() *EvmDBConfig {
	return &EvmDBConfig{
		DBName:        "evm",
		DBBackend:     "goleveldb",
		CacheSizeMegs: 20,
	}
}

// Clone returns a deep clone of the config.
func (cfg *EvmDBConfig) Clone() *EvmDBConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}
