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
	// Specifies the number of IAVL tree versions that should be kept in memory before writing a new
	// version to disk.
	// If set to zero every version will be written to disk unless overridden via the on-chain config.
	// If set to -1 every version will always be written to disk, regardless of the on-chain config.
	IAVLFlushInterval int64
	// Enables caching of the mutable app store in tx handlers & middleware.
	CachingEnabled bool
}

func DefaultConfig() *AppStoreConfig {
	return &AppStoreConfig{
		Version:            3,
		CompactOnLoad:      false,
		MaxVersions:        0,
		PruneInterval:      0,
		PruneBatchSize:     50,
		SaveEVMStateToIAVL: false,
		IAVLFlushInterval:  0, // allow override via on-chain config
		CachingEnabled:     false,
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
