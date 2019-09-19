package evmaux

type EvmAuxStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
	// MaxReceipts defines the maximum number of EVM tx receipts stored in EVM auxiliary store
	MaxReceipts uint64
}

func DefaultEvmAuxStoreConfig() *EvmAuxStoreConfig {
	return &EvmAuxStoreConfig{
		DBName:      "evmaux",
		DBBackend:   "goleveldb",
		MaxReceipts: 2000,
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
