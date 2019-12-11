package evm

type EvmStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
	// CacheSizeMegs defines cache size (in megabytes) of EVM store
	CacheSizeMegs int
	// WriteBufferMegs for the goleveldb backend EVM Store
	WriteBufferMegs int
	// NumCachedRoots defines a number of in-memory cached EVM roots
	NumCachedRoots int
	// Specifies the number of Merkle tree versions that should be kept in memory before writing a
	// new version to disk.
	// If set to zero every version will be written to disk unless overridden via the on-chain config
	// AppStore.IAVLFlushInterval setting.
	// If set to -1 every version will always be written to disk, regardless of the on-chain config.
	FlushInterval int64
}

func DefaultEvmStoreConfig() *EvmStoreConfig {
	return &EvmStoreConfig{
		DBName:          "evm",
		DBBackend:       "goleveldb",
		CacheSizeMegs:   256,
		WriteBufferMegs: 4,
		NumCachedRoots:  500,
		FlushInterval:   0,
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
