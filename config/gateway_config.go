// +build gateway

package config

import (
	"github.com/loomnetwork/transfer-gateway/gateway"
	dpos2cfg "github.com/loomnetwork/transfer-gateway/oracles/dpos2/config"
)

type TransferGatewayConfig = gateway.TransferGatewayConfig
type OracleSerializableConfig = dpos2cfg.OracleSerializableConfig

func DefaultTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return gateway.DefaultConfig(rpcProxyPort)
}

func DefaultLoomCoinTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return gateway.DefaultLoomCoinTGConfig(rpcProxyPort)
}

func DefaultTronTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return gateway.DefaultTronConfig(rpcProxyPort)
}

func DefaultBinanceTGConfig() *TransferGatewayConfig {
	return &TransferGatewayConfig{
		ContractEnabled: true, // enabled by default for easier production deployment
	}
}

func DefaultDPOS2OracleConfig() *OracleSerializableConfig {
	return &OracleSerializableConfig{}
}

const transferGatewayLoomYamlTemplate = `
#
# Transfer Gateway
#
TransferGateway:
  # Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
  ContractEnabled: {{ .TransferGateway.ContractEnabled }}
  # Enables the in-process Transfer Gateway Oracle.
  # If this is enabled ContractEnabled must be set to true.
  OracleEnabled: {{ .TransferGateway.OracleEnabled }}
  # URI of Ethereum node the Oracle should connect to, and retrieve Mainnet events from.
  EthereumURI: "{{ .TransferGateway.EthereumURI }}"
  # Address of Transfer Gateway contract on Mainnet
  # e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
  MainnetContractHexAddress: "{{ .TransferGateway.MainnetContractHexAddress }}"
  # Path to Ethereum private key on disk that should be used by the Oracle to sign withdrawals,
  # can be a relative, or absolute path
  MainnetPrivateKeyPath: "{{ .TransferGateway.MainnetPrivateKeyPath }}"
  # Path to DAppChain private key on disk that should be used by the Oracle to sign txs send to
  # the DAppChain Transfer Gateway contract
  DAppChainPrivateKeyPath: "{{ .TransferGateway.DAppChainPrivateKeyPath }}"
  DAppChainReadURI: "{{ .TransferGateway.DAppChainReadURI }}"
  DAppChainWriteURI: "{{ .TransferGateway.DAppChainWriteURI }}"
  # Websocket URI that should be used to subscribe to DAppChain events (only used for tests)
  DAppChainEventsURI: "{{ .TransferGateway.DAppChainEventsURI }}"
  DAppChainPollInterval: {{ .TransferGateway.DAppChainPollInterval }}
  MainnetPollInterval: {{ .TransferGateway.MainnetPollInterval }}
  # Oracle log verbosity (debug, info, error, etc.)
  OracleLogLevel: "{{ .TransferGateway.OracleLogLevel }}"
  OracleLogDestination: "{{ .TransferGateway.OracleLogDestination }}"
  # Number of seconds to wait before starting the Oracle.
  OracleStartupDelay: {{ .TransferGateway.OracleStartupDelay }}
  # Number of seconds to wait between reconnection attempts.
  OracleReconnectInterval: {{ .TransferGateway.OracleReconnectInterval }}
  # Address on from which the out-of-process Oracle should expose the status & metrics endpoints.
  OracleQueryAddress: "{{ .TransferGateway.OracleQueryAddress }}"
  {{if .TransferGateway.BatchSignFnConfig -}}
  BatchSignFnConfig:
    Enabled: {{ .TransferGateway.BatchSignFnConfig.Enabled }}
    LogLevel: "{{ .TransferGateway.BatchSignFnConfig.LogLevel }}"
    LogDestination: "{{ .TransferGateway.BatchSignFnConfig.LogDestination }}"
    MainnetPrivateKeyPath: "{{ .TransferGateway.BatchSignFnConfig.MainnetPrivateKeyPath }}"
    MainnetPrivateKeyHsmEnabled: "{{ .TransferGateway.BatchSignFnConfig.MainnetPrivateKeyHsmEnabled }}"
  {{- end}}

#
# Loomcoin Transfer Gateway
#
LoomCoinTransferGateway:
  # Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
  ContractEnabled: {{ .LoomCoinTransferGateway.ContractEnabled }}
  # Enables the in-process Transfer Gateway Oracle.
  # If this is enabled ContractEnabled must be set to true.
  OracleEnabled: {{ .LoomCoinTransferGateway.OracleEnabled }}
  # URI of Ethereum node the Oracle should connect to, and retrieve Mainnet events from.
  EthereumURI: "{{ .LoomCoinTransferGateway.EthereumURI }}"
  # Address of Transfer Gateway contract on Mainnet
  # e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
  MainnetContractHexAddress: "{{ .LoomCoinTransferGateway.MainnetContractHexAddress }}"
  # Path to Ethereum private key on disk that should be used by the Oracle to sign withdrawals,
  # can be a relative, or absolute path
  MainnetPrivateKeyPath: "{{ .LoomCoinTransferGateway.MainnetPrivateKeyPath }}"
  # Path to DAppChain private key on disk that should be used by the Oracle to sign txs send to
  # the DAppChain Transfer Gateway contract
  DAppChainPrivateKeyPath: "{{ .LoomCoinTransferGateway.DAppChainPrivateKeyPath }}"
  DAppChainReadURI: "{{ .LoomCoinTransferGateway.DAppChainReadURI }}"
  DAppChainWriteURI: "{{ .LoomCoinTransferGateway.DAppChainWriteURI }}"
  # Websocket URI that should be used to subscribe to DAppChain events (only used for tests)
  DAppChainEventsURI: "{{ .LoomCoinTransferGateway.DAppChainEventsURI }}"
  DAppChainPollInterval: {{ .LoomCoinTransferGateway.DAppChainPollInterval }}
  MainnetPollInterval: {{ .LoomCoinTransferGateway.MainnetPollInterval }}
  # Oracle log verbosity (debug, info, error, etc.)
  OracleLogLevel: "{{ .LoomCoinTransferGateway.OracleLogLevel }}"
  OracleLogDestination: "{{ .LoomCoinTransferGateway.OracleLogDestination }}"
  # Number of seconds to wait before starting the Oracle.
  OracleStartupDelay: {{ .LoomCoinTransferGateway.OracleStartupDelay }}
  # Number of seconds to wait between reconnection attempts.
  OracleReconnectInterval: {{ .LoomCoinTransferGateway.OracleReconnectInterval }}
  # Address on from which the out-of-process Oracle should expose the status & metrics endpoints.
  OracleQueryAddress: "{{ .LoomCoinTransferGateway.OracleQueryAddress }}"
  {{if .LoomCoinTransferGateway.BatchSignFnConfig -}}
  BatchSignFnConfig:
    Enabled: {{ .LoomCoinTransferGateway.BatchSignFnConfig.Enabled }}
    LogLevel: "{{ .LoomCoinTransferGateway.BatchSignFnConfig.LogLevel }}"
    LogDestination: "{{ .LoomCoinTransferGateway.BatchSignFnConfig.LogDestination }}"
    MainnetPrivateKeyPath: "{{ .LoomCoinTransferGateway.BatchSignFnConfig.MainnetPrivateKeyPath }}"
    MainnetPrivateKeyHsmEnabled: "{{ .LoomCoinTransferGateway.BatchSignFnConfig.MainnetPrivateKeyHsmEnabled }}"
  {{- end}}

#
# Tron Transfer Gateway
#
TronTransferGateway:
  # Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
  ContractEnabled: {{ .TronTransferGateway.ContractEnabled }}
  # Enables the in-process Transfer Gateway Oracle.
  # If this is enabled ContractEnabled must be set to true.
  OracleEnabled: {{ .TronTransferGateway.OracleEnabled }}
  # URI of Tron node the Oracle should connect to, and retrieve Mainnet events from.
  TronURI: "{{ .TronTransferGateway.TronURI }}"
  # Address of Transfer Gateway contract on Mainnet
  # e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
  MainnetContractHexAddress: "{{ .TronTransferGateway.MainnetContractHexAddress }}"
  # Path to Ethereum private key on disk that should be used by the Oracle to sign withdrawals,
  # can be a relative, or absolute path
  MainnetPrivateKeyPath: "{{ .TronTransferGateway.MainnetPrivateKeyPath }}"
  # Path to DAppChain private key on disk that should be used by the Oracle to sign txs send to
  # the DAppChain Transfer Gateway contract
  DAppChainPrivateKeyPath: "{{ .TronTransferGateway.DAppChainPrivateKeyPath }}"
  DAppChainReadURI: "{{ .TronTransferGateway.DAppChainReadURI }}"
  DAppChainWriteURI: "{{ .TronTransferGateway.DAppChainWriteURI }}"
  # Websocket URI that should be used to subscribe to DAppChain events (only used for tests)
  DAppChainEventsURI: "{{ .TronTransferGateway.DAppChainEventsURI }}"
  DAppChainPollInterval: {{ .TronTransferGateway.DAppChainPollInterval }}
  MainnetPollInterval: {{ .TronTransferGateway.MainnetPollInterval }}
  # Oracle log verbosity (debug, info, error, etc.)
  OracleLogLevel: "{{ .TronTransferGateway.OracleLogLevel }}"
  OracleLogDestination: "{{ .TronTransferGateway.OracleLogDestination }}"
  # Number of seconds to wait before starting the Oracle.
  OracleStartupDelay: {{ .TronTransferGateway.OracleStartupDelay }}
  # Number of seconds to wait between reconnection attempts.
  OracleReconnectInterval: {{ .TronTransferGateway.OracleReconnectInterval }}
  # Address on from which the out-of-process Oracle should expose the status & metrics endpoints.
  OracleQueryAddress: "{{ .TronTransferGateway.OracleQueryAddress }}"
  {{if .TronTransferGateway.BatchSignFnConfig -}}
  BatchSignFnConfig:
    Enabled: {{ .TronTransferGateway.BatchSignFnConfig.Enabled }}
    LogLevel: "{{ .TronTransferGateway.BatchSignFnConfig.LogLevel }}"
    LogDestination: "{{ .TronTransferGateway.BatchSignFnConfig.LogDestination }}"
    MainnetPrivateKeyPath: "{{ .TronTransferGateway.BatchSignFnConfig.MainnetPrivateKeyPath }}"
    MainnetPrivateKeyHsmEnabled: "{{ .TronTransferGateway.BatchSignFnConfig.MainnetPrivateKeyHsmEnabled }}"
  {{- end}}

#
# Binance Transfer Gateway
#
BinanceTransferGateway:
  # Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
  ContractEnabled: {{ .BinanceTransferGateway.ContractEnabled }}

BinanceSmartchainTransferGateway:
  # Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
  ContractEnabled: {{ .BinanceSmartchainTransferGateway.ContractEnabled }}


#
# Oracle serializable 
#
DPOSv2OracleConfig:
  Enabled: {{ .DPOSv2OracleConfig.Enabled }}
  StatusServiceAddress: "{{ .DPOSv2OracleConfig.StatusServiceAddress }}"
  MainnetPollInterval: {{ .DPOSv2OracleConfig.MainnetPollInterval }}
  {{if .DPOSv2OracleConfig.DAppChainCfg -}}
    DAppChainCfg: 
      WriteURI: "{{ .DPOSv2OracleConfig.DAppChainCfg.WriteURI }}"
      ReadURI: "{{ .DPOSv2OracleConfig.DAppChainCfg.ReadURI }}"
      PrivateKeyPath: "{{ .DPOSv2OracleConfig.DAppChainCfg.PrivateKeyPath }}"
  {{end}}
  {{if .DPOSv2OracleConfig.EthClientCfg -}}
    EthClientCfg: 
      EthereumURI: "{{ .DPOSv2OracleConfig.EthClientCfg.EthereumURI }}"
      PrivateKeyPath: {{ .DPOSv2OracleConfig.EthClientCfg.PrivateKeyPath }}
  {{end}}
  {{if .DPOSv2OracleConfig.TimeLockWorkerCfg -}}
    TimeLockWorkerCfg: 
      TimeLockFactoryHexAddress: "{{ .DPOSv2OracleConfig.TimeLockWorkerCfg.TimeLockFactoryHexAddress }}"
      Enabled: {{ .DPOSv2OracleConfig.TimeLockWorkerCfg.Enabled }}
  {{end}}
`
