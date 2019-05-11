package loomchain

import (
	"fmt"
	"strconv"

	loom "github.com/loomnetwork/go-loom"
)

const (
	dposFeeFloor = "dpos.feeFloor"
)

type Config struct {
	cfg  map[string]string
	dpos loom.DPOSConfig
}

func NewChainConfig(cfg map[string]string) loom.Config {
	return &Config{
		cfg: cfg,
		dpos: &DPOSConfig{
			cfg: cfg,
		},
	}
}

func (cfg *Config) DPOS() loom.DPOSConfig {
	return cfg.dpos
}

func (cfg *Config) GetConfig(key string) string {
	return cfg.cfg[key]
}

type DPOSConfig struct {
	cfg map[string]string
}

func (dpos *DPOSConfig) FeeFloor(val int64) int64 {
	feeFloor, err := getInt64(dposFeeFloor, dpos.cfg)
	if err != nil {
		return val
	}
	return feeFloor
}

func getInt64(key string, cfg map[string]string) (int64, error) {
	if value, ok := cfg[key]; ok {
		v, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return v, nil
		}
		return 0, err
	}
	return 0, fmt.Errorf("Key %s not found", key)
}
