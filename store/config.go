package store

type AppStoreConfig struct {
	// 1 - IAVL, 2 - MultiReaderIAVL, defaults to 1
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
	// DB backend to use for storing a materialized view of the latest persistent app state
	// possible values are "none" | "goleveldb". Defaults to "none", which means the
	// values are stored in app.db
	LatestStateDBBackend string
	LatestStateDBName    string
}

func DefaultConfig() *AppStoreConfig {
	return &AppStoreConfig{
		Version:              1,
		CompactOnLoad:        false,
		MaxVersions:          0,
		PruneInterval:        0,
		PruneBatchSize:       50,
		LatestStateDBBackend: "goleveldb",
		LatestStateDBName:    "app_state",
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
