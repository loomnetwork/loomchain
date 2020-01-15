// +build evm

package debug

import (
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
)

type JsonTraceConfig struct {
	LogConfig *JsonLogConfig `json:"logconfig,omitempty"`
	Tracer    string         `json:"tracer,omitempty"`
	Timeout   string         `json:"address,omitempty"`
}

type JsonLogConfig struct {
	DisableStorage bool `json:"disableStorage,omitempty"`
	DisableMemory  bool `json:"disableMemory,omitempty"`
	DisableStack   bool `json:"disableStack,omitempty"`
}

type JsonStorageRangeResult struct {
	eth.StorageRangeResult
	Complete bool `json:"complete"`
}

func DecTraceConfig(jcfg *JsonTraceConfig) eth.TraceConfig {
	var logConfig *vm.LogConfig
	if jcfg == nil {
		return eth.TraceConfig{
			LogConfig: logConfig,
			Tracer:    nil,
			Timeout:   nil,
			Reexec:    nil,
		}
	}
	if jcfg.LogConfig != nil {
		logConfig = &vm.LogConfig{
			DisableMemory:  jcfg.LogConfig.DisableMemory,
			DisableStack:   jcfg.LogConfig.DisableStack,
			DisableStorage: jcfg.LogConfig.DisableStorage,
		}
	}
	return eth.TraceConfig{
		LogConfig: logConfig,
		Tracer:    &jcfg.Tracer,
		Timeout:   &jcfg.Timeout,
		Reexec:    nil,
	}
}
