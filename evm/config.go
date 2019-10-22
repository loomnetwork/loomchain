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
}

func DefaultEvmStoreConfig() *EvmStoreConfig {
	return &EvmStoreConfig{
		DBName:          "evm",
		DBBackend:       "goleveldb",
		CacheSizeMegs:   256,
		WriteBufferMegs: 4,
		NumCachedRoots:  100,
	}
}

type EvmConfig struct {
	// MaxBlockLimit defines the maximum number of blocks that can be queried in a request
	MaxBlockLimit int64
}

func DefaultEvmConfig() *EvmConfig {
	return &EvmConfig{
		MaxBlockLimit: 20,
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
