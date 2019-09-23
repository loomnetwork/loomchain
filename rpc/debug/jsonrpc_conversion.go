// +build evm

package debug

import (
	"github.com/ethereum/go-ethereum/core/vm"
	etheth "github.com/ethereum/go-ethereum/eth"
)

type JsonTraceConfig struct {
	DisableStorage bool   `json:"disableStorage,omitempty"`
	DisableMemory  bool   `json:"disableMemory,omitempty"`
	DisableStack   bool   `json:"disableStack,omitempty"`
	Tracer         string `json:"tracer,omitempty"`
	Timeout        string `json:"address,omitempty"`
}

func DecTraceConfig(jcfg JsonTraceConfig) etheth.TraceConfig {
	return etheth.TraceConfig{
		LogConfig: &vm.LogConfig{
			DisableMemory:  jcfg.DisableMemory,
			DisableStack:   jcfg.DisableStack,
			DisableStorage: jcfg.DisableStorage,
		},
		Tracer:  &jcfg.Tracer,
		Timeout: &jcfg.Timeout,
		Reexec:  nil,
	}
}
