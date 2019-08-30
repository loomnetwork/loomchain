package store

import (
	"github.com/gogo/protobuf/proto"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/config"
)

const configKey = "config"

// LoadOnChainConfig loads the on-chain config from the given kv store.
func LoadOnChainConfig(kvStore KVReader) (*cctypes.Config, error) {
	configBytes := kvStore.Get([]byte(configKey))
	cfg := config.DefaultConfig()
	if len(configBytes) > 0 {
		if err := proto.UnmarshalMerge(configBytes, cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// SaveOnChainConfig saves an on-chain config to the given kv store.
func SaveOnChainConfig(kvStore KVWriter, cfg *cctypes.Config) error {
	configBytes, err := proto.Marshal(cfg)
	if err != nil {
		return err
	}
	kvStore.Set([]byte(configKey), configBytes)
	return nil
}
