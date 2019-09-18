package debug

type JsonTraceConfig struct {
	DisableStorage bool   `json:"disableStorage,omitempty"`
	DisableMemory  bool   `json:"disableMemory,omitempty"`
	DisableStack   bool   `json:"disableStack,omitempty"`
	Tracer         string `json:"tracer,omitempty"`
	Timeout        string `json:"address,omitempty"`
}
