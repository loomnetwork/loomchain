// +build !evm

package oracle

import "errors"

type Config struct {
	Enabled           bool
	TimeLockWorkerCfg TimeLockWorkerConfig
}

type TimeLockWorkerConfig struct {
	Enabled bool
}

type Oracle struct {
}

func (o *Oracle) Init() error {
	return errors.New("not implemented in non-EVM build")
}

func NewOracle(cfg *Config) *Oracle {
	return nil
}

func (o *Oracle) Run() {

}
