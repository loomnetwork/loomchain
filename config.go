package loomchain

import loom "github.com/loomnetwork/go-loom"

type Config struct {
	dpos loom.DPOS
}

func NewChainConfig(cfg map[string]string) loom.Config {
	for key, value := range cfg {
		// doing some mapping
		_ = key
		_ = value
	}

	return &Config{}
}

func (cfg *Config) DPOS() loom.DPOS {
	return cfg.dpos
}

type DPOS struct {
	feeFloor int64
}

func (dpos *DPOS) FeeFloor(val int64) int64 {
	return dpos.feeFloor
}
