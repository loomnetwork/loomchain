package store

type AppStoreConfig struct {
	// 1 - IAVL, 2 - MultiReaderIAVL, 3 - MultiWriterAppStore, defaults to 1
	Version int64
	// If true the app store will be compacted before it's loaded to reclaim disk space.
	CompactOnLoad bool
	// Maximum number of app store versions to keep, if zero old versions will never be deleted.
	MaxVersions int64
	// Number of seconds to wait after pruning a batch of old versions from the app store.
	// If this is set to zero the app store will only be pruned after a new version is saved.
	PruneInterval int64
	// Number of versions to prune at a time.
	PruneBatchSize int64

	// If true the app store will write EVM state to both IAVLStore and EvmStore
	// This config works with AppStore Version 3 (MultiWriterAppStore) only
	SaveEVMStateToIAVL bool

	IAVLFlushInterval int64
}

func DefaultConfig() *AppStoreConfig {
	return &AppStoreConfig{
		Version:            3,
		CompactOnLoad:      false,
		MaxVersions:        0,
		PruneInterval:      0,
		PruneBatchSize:     50,
		SaveEVMStateToIAVL: false,
		IAVLFlushInterval:  0, //default to zero until we know its ready
	}
}

// Clone returns a deep clone of the config.
func (c *AppStoreConfig) Clone() *AppStoreConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

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

// Clone returns a deep clone of the config.
func (cfg *EvmStoreConfig) Clone() *EvmStoreConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}
