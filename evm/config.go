package evm

type EVMStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend EVM store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
}

func DefaultEVMStoreConfig() *EVMStoreConfig {
	return &EVMStoreConfig{
		DBName:    "evm",
		DBBackend: "goleveldb",
	}
}

// Clone returns a deep clone of the config.
func (evmStore *EVMStoreConfig) Clone() *EVMStoreConfig {
	if evmStore == nil {
		return nil
	}
	clone := *evmStore
	return &clone
}
