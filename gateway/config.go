package gateway

import (
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
)

type BatchWithdrawalSignFnConfig struct {
	Enabled                     bool
	LogLevel                    string
	LogDestination              string
	MainnetPrivateKeyPath       string
	MainnetPrivateKeyHsmEnabled bool
}

type WithdrawalSigType int

const (
	UnprefixedWithdrawalSigType WithdrawalSigType = 1
	PrefixedWithdrawalSigType   WithdrawalSigType = 2
)

type TransferGatewayConfig struct {
	// Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
	ContractEnabled bool
	// Loads the Unsafe gateway methods
	Unsafe bool
	// Specifies which signing function to use for the gateway
	WithdrawalSig WithdrawalSigType
	// Enables the in-process Transfer Gateway Oracle.
	// If this is enabled ContractEnabled must be set to true.
	OracleEnabled bool
	// URI of Ethereum node the Oracle should connect to, and retrieve Mainnet events from.
	EthereumURI string
	// Address of Transfer Gateway contract on Mainnet
	// e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
	MainnetContractHexAddress string
	// Path to Ethereum private key on disk or YubiHSM that should be used by the Oracle to sign withdrawals,
	// can be a relative, or absolute path
	MainnetPrivateKeyHsmEnabled bool
	MainnetPrivateKeyPath       string
	// Path to DAppChain private key on disk that should be used by the Oracle to sign txs send to
	// the DAppChain Transfer Gateway contract
	DappChainPrivateKeyHsmEnabled bool
	DAppChainPrivateKeyPath       string
	DAppChainReadURI              string
	DAppChainWriteURI             string
	// Websocket URI that should be used to subscribe to DAppChain events (only used for tests)
	DAppChainEventsURI    string
	DAppChainPollInterval int
	MainnetPollInterval   int
	// Number of Ethereum block confirmations the Oracle should wait for before forwarding events
	// from the Ethereum Gateway contract to the DAppChain Gateway contract.
	NumMainnetBlockConfirmations int
	// Oracle log verbosity (debug, info, error, etc.)
	OracleLogLevel       string
	OracleLogDestination string
	// Number of seconds to wait before starting the Oracle.
	OracleStartupDelay int32
	// Number of seconds to wait between reconnection attempts.
	OracleReconnectInterval int32
	// Address on from which the out-of-process Oracle should expose the status & metrics endpoints.
	OracleQueryAddress string

	BatchSignFnConfig *BatchWithdrawalSignFnConfig

	// List of DAppChain addresses that aren't allowed to withdraw to the Mainnet Gateway
	WithdrawerAddressBlacklist []string
}

func DefaultConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{
		ContractEnabled:               false,
		Unsafe:                        false,
		OracleEnabled:                 false,
		EthereumURI:                   "ws://127.0.0.1:8545",
		MainnetContractHexAddress:     "",
		MainnetPrivateKeyHsmEnabled:   false,
		MainnetPrivateKeyPath:         "",
		DappChainPrivateKeyHsmEnabled: false,
		DAppChainPrivateKeyPath:       "",
		DAppChainReadURI:              fmt.Sprintf("http://127.0.0.1:%d/query", rpcProxyPort),
		DAppChainWriteURI:             fmt.Sprintf("http://127.0.0.1:%d/rpc", rpcProxyPort),
		DAppChainEventsURI:            fmt.Sprintf("ws://127.0.0.1:%d/queryws", rpcProxyPort),
		DAppChainPollInterval:         10,
		MainnetPollInterval:           10,
		NumMainnetBlockConfirmations:  15,
		OracleLogLevel:                "info",
		OracleLogDestination:          "file://tgoracle.log",
		OracleStartupDelay:            5,
		OracleQueryAddress:            "127.0.0.1:9998",
		BatchSignFnConfig: &BatchWithdrawalSignFnConfig{
			Enabled:                     false,
			LogLevel:                    "info",
			LogDestination:              "file://-",
			MainnetPrivateKeyPath:       "",
			MainnetPrivateKeyHsmEnabled: false,
		},
		WithdrawalSig: UnprefixedWithdrawalSigType,
	}
}

func DefaultLoomCoinTGConfig(rpcProxyPort int32) *TransferGatewayConfig {
	return &TransferGatewayConfig{
		ContractEnabled:               false,
		Unsafe:                        false,
		OracleEnabled:                 false,
		EthereumURI:                   "ws://127.0.0.1:8545",
		MainnetContractHexAddress:     "",
		MainnetPrivateKeyHsmEnabled:   false,
		MainnetPrivateKeyPath:         "",
		DappChainPrivateKeyHsmEnabled: false,
		DAppChainPrivateKeyPath:       "",
		DAppChainReadURI:              fmt.Sprintf("http://127.0.0.1:%d/query", rpcProxyPort),
		DAppChainWriteURI:             fmt.Sprintf("http://127.0.0.1:%d/rpc", rpcProxyPort),
		DAppChainEventsURI:            fmt.Sprintf("ws://127.0.0.1:%d/queryws", rpcProxyPort),
		DAppChainPollInterval:         10,
		MainnetPollInterval:           10,
		NumMainnetBlockConfirmations:  15,
		OracleLogLevel:                "info",
		OracleLogDestination:          "file://loomcoin_tgoracle.log",
		OracleStartupDelay:            5,
		OracleQueryAddress:            "127.0.0.1:9997",
		BatchSignFnConfig: &BatchWithdrawalSignFnConfig{
			Enabled:                     false,
			LogLevel:                    "info",
			LogDestination:              "file://-",
			MainnetPrivateKeyPath:       "",
			MainnetPrivateKeyHsmEnabled: false,
		},
		WithdrawalSig: UnprefixedWithdrawalSigType,
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

// Validate does a basic sanity check of the config.
func (c *TransferGatewayConfig) Validate() error {
	if c.NumMainnetBlockConfirmations < 0 {
		return errors.New("NumMainnetBlockConfirmations can't be negative")
	}
	return nil
}

func (c *TransferGatewayConfig) GetWithdrawerAddressBlacklist() ([]loom.Address, error) {
	//var addrList []loom.Address
	addrList := make([]loom.Address, 0, len(c.WithdrawerAddressBlacklist))
	for _, addrStr := range c.WithdrawerAddressBlacklist {
		addr, err := loom.ParseAddress(addrStr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse address in WithdrawerAddressBlacklist")
		}
		addrList = append(addrList, addr)
	}
	return addrList, nil
}
