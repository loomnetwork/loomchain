package store

type AppStoreConfig struct {
	// If true the app store will be compacted before it's loaded to reclaim disk space.
	CompactOnLoad bool
	// Maximum number of app store versions to keep, if zero old versions will never be deleted.
	MaxVersions int64
	// Number of seconds to wait after pruning a batch of old versions from the app store.
	// If this is set to zero the app store will only be pruned after a new version is saved.
	PruneInterval int64
	// Number of versions to prune at a time.
	PruneBatchSize int64
}

func DefaultConfig() *AppStoreConfig {
	return &AppStoreConfig{
		CompactOnLoad:  true,
		MaxVersions:    50,
		PruneInterval:  0,
		PruneBatchSize: 50,
	}
}
