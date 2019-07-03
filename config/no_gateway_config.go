// +build !gateway

package config

type TransferGatewayConfig struct {
	ContractEnabled bool
}

func DefaultTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{}
}

func DefaultLoomCoinTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{}
}

func DefaultTronTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{}
}

func DefaultBinanceTGConfig() *TransferGatewayConfig {
	return &TransferGatewayConfig{}
}

// Clone returns a deep clone of the config.
func (c *TransferGatewayConfig) Clone() *TransferGatewayConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type OracleSerializableConfig struct {
}

func DefaultDPOS2OracleConfig() *OracleSerializableConfig {
	return &OracleSerializableConfig{}
}

const transferGatewayLoomYamlTemplate = ""
