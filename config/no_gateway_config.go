// +build !gateway

package config

type TransferGatewayConfig struct {
	// Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
	ContractEnabled bool
	// Loads the Unsafe gateway methods
	Unsafe bool
}

func DefaultTransferGatewayConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{
		ContractEnabled: false,
		Unsafe:          false,
	}
}

// Clone returns a deep clone of the config.
func (c *TransferGatewayConfig) Clone() *TransferGatewayConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

const transferGatewayLoomYamlTemplate = ""
