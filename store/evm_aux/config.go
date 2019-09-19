package evmaux

type EvmAuxStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
	// Persist data in database
	PersistData bool
}

func DefaultEvmAuxStoreConfig() *EvmAuxStoreConfig {
	return &EvmAuxStoreConfig{
		DBName:      "evmaux",
		DBBackend:   "goleveldb",
		PersistData: true,
	}
}

// Clone returns a deep clone of the config.
func (cfg *EvmAuxStoreConfig) Clone() *EvmAuxStoreConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}
