package gateway

import (
	"fmt"
)

type TransferGatewayConfig struct {
	// Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
	ContractEnabled bool
	// Enables the in-process Transfer Gateway Oracle.
	// If this is enabled ContractEnabled must be set to true.
	OracleEnabled bool
	// URI of Ethereum node the Oracle should connect to, and retrieve Mainnet events from.
	EthereumURI string
	// Address of Transfer Gateway contract on Mainnet
	// e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
	MainnetContractHexAddress string
	// Path to Ethereum private key on disk that should be used by the Oracle to sign withdrawals,
	// can be a relative, or absolute path
	MainnetPrivateKeyPath string
	// Path to DAppChain private key on disk that should be used by the Oracle to sign txs send to
	// the DAppChain Transfer Gateway contract
	DAppChainPrivateKeyPath string
	DAppChainReadURI        string
	DAppChainWriteURI       string
	// Websocket URI that should be used to subscribe to DAppChain events (only used for tests)
	DAppChainEventsURI    string
	DAppChainPollInterval int
	MainnetPollInterval   int
	// Oracle log verbosity (debug, info, error, etc.)
	OracleLogLevel       string
	OracleLogDestination string
	// Number of seconds to wait before starting the Oracle.
	OracleStartupDelay int32
	// Number of seconds to wait between reconnection attempts.
	OracleReconnectInterval int32
}

func DefaultConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{
		ContractEnabled:           false,
		OracleEnabled:             false,
		EthereumURI:               "ws://127.0.0.1:8545",
		MainnetContractHexAddress: "",
		MainnetPrivateKeyPath:     "",
		DAppChainPrivateKeyPath:   "",
		DAppChainReadURI:          fmt.Sprintf("http://127.0.0.1:%d/query", rpcProxyPort),
		DAppChainWriteURI:         fmt.Sprintf("http://127.0.0.1:%d/rpc", rpcProxyPort),
		DAppChainEventsURI:        fmt.Sprintf("ws://127.0.0.1:%d/queryws", rpcProxyPort),
		DAppChainPollInterval:     10,
		MainnetPollInterval:       10,
		OracleLogLevel:            "info",
		OracleLogDestination:      "file://tgoracle.log",
		OracleStartupDelay:        5,
	}
}
