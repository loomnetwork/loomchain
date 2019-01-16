package config

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	dposv2OracleCfg "github.com/loomnetwork/loomchain/builtin/plugins/dposv2/oracle/config"
	plasmacfg "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/config"
	"github.com/loomnetwork/loomchain/gateway"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	receipts "github.com/loomnetwork/loomchain/receipts/handler"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/spf13/viper"

	"github.com/loomnetwork/loomchain/db"
)

type Config struct {
	RootDir            string
	DBName             string
	DBBackend          string
	GenesisFile        string
	PluginsDir         string
	QueryServerHost    string
	EventDispatcherURI string
	ContractLogLevel   string
	LogDestination     string
	LoomLogLevel       string
	BlockchainLogLevel string
	Peers              string
	PersistentPeers    string
	RPCListenAddress   string
	ChainID            string
	RPCProxyPort       int32
	RPCBindAddress     string
	// Controls whether or not empty blocks should be generated periodically if there are no txs or
	// AppHash changes. Defaults to true.
	CreateEmptyBlocks          bool
	SessionMaxAccessCount      int64
	SessionDuration            int64
	LogStateDB                 bool
	LogEthDbBatch              bool
	UseCheckTx                 bool
	RegistryVersion            int32
	ReceiptsVersion            int32
	EVMPersistentTxReceiptsMax uint64
	TransferGateway            *gateway.TransferGatewayConfig
	LoomCoinTransferGateway    *gateway.TransferGatewayConfig
	PlasmaCash                 *plasmacfg.PlasmaCashSerializableConfig
	// When this setting is enabled Loom EVM accounts are hooked up to the builtin ethcoin Go contract,
	// which makes it possible to use the payable/transfer features of the EVM to transfer ETH in
	// Solidity contracts running on the Loom EVM. This setting is disabled by default, which means
	// all the EVM accounts always have a zero balance.
	EVMAccountsEnabled bool
	EVMDebugEnabled    bool
	BootLegacyDPoS     bool

	Oracle              string
	DeployEnabled       bool
	CallEnabled         bool
	CallSessionDuration int64
	Karma 				*KarmaConfig
	DPOSVersion         int64

	CachingStoreConfig *store.CachingStoreConfig

	DPOSv2OracleConfig *dposv2OracleCfg.OracleSerializableConfig

	AppStore  *store.AppStoreConfig
	HsmConfig *hsmpv.HsmConfig
	TxLimiter *throttle.TxLimiterConfig
	Metrics   *Metrics
}

type Metrics struct {
	EventHandling bool
}

type KarmaConfig struct {
	Enabled bool    // Activate karma module
	ContractEnabled bool    // Allows you to deploy karma contract to collect data even if chain doesn't use it
	MaxCallCount    int64   // Maximum number call transactions per session duration
	SessionDuration int64   // Session length in seconds
}

func DefaultMetrics() *Metrics {
	return &Metrics{
		EventHandling: true,
	}
}

func DefaultKarmaConfig() *KarmaConfig {
	return &KarmaConfig{
		Enabled:         false,
		ContractEnabled: false,
		MaxCallCount:    0,
		SessionDuration: 0,
	}
}

type ContractConfig struct {
	VMTypeName string          `json:"vm"`
	Format     string          `json:"format,omitempty"`
	Name       string          `json:"name,omitempty"`
	Location   string          `json:"location"`
	Init       json.RawMessage `json:"init"`
}

func (c ContractConfig) VMType() vm.VMType {
	return vm.VMType(vm.VMType_value[c.VMTypeName])
}

type Genesis struct {
	Contracts []ContractConfig `json:"contracts"`
}

//Structure for LOOM ENV

type Env struct {
	Version         string `json:"version"`
	Build           string `json:"build"`
	BuildVariant    string `json:"buildvariant"`
	GitSha          string `json:"gitsha"`
	GoLoom          string `json:"goloom"`
	GoEthereum      string `json:"goethereum"`
	GoPlugin        string `json:"goplugin"`
	PluginPath      string `json:"pluginpath"`
	QueryServerHost string `json:"queryserverhost"`
	Peers           string `json:"peers"`
}

//Structure for Loom ENVINFO - ENV + Genesis + Loom.yaml

type EnvInfo struct {
	Env         Env     `json:"env"`
	LoomGenesis Genesis `json:"loomGenesis"`
	LoomConfig  Config  `json:"loomConfig"`
}

func ParseConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("LOOM")

	v.SetConfigName("loom")                        // name of config file (without extension)
	v.AddConfigPath("./")                          // search root directory
	v.AddConfigPath(filepath.Join("./", "config")) // search root directory /config
	v.AddConfigPath("./../../../")

	v.ReadInConfig()
	conf := DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}

func ReadGenesis(path string) (*Genesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(file)

	var gen Genesis
	err = dec.Decode(&gen)
	if err != nil {
		return nil, err
	}

	return &gen, nil
}

func DefaultConfig() *Config {
	cfg := &Config{
		RootDir:                    ".",
		DBName:                     "app",
		DBBackend:                  db.GoLevelDBBackend,
		GenesisFile:                "genesis.json",
		PluginsDir:                 "contracts",
		QueryServerHost:            "tcp://127.0.0.1:9999",
		RPCListenAddress:           "tcp://0.0.0.0:46657", //TODO this is an ephemeral port in linux, we should move this
		EventDispatcherURI:         "",
		ContractLogLevel:           "info",
		LoomLogLevel:               "info",
		LogDestination:             "",
		BlockchainLogLevel:         "error",
		Peers:                      "",
		PersistentPeers:            "",
		ChainID:                    "",
		RPCProxyPort:               46658,
		RPCBindAddress:             "tcp://0.0.0.0:46658",
		CreateEmptyBlocks:          true,
		LogStateDB:                 false,
		LogEthDbBatch:              false,
		UseCheckTx:                 true,
		RegistryVersion:            int32(registry.RegistryV1),
		ReceiptsVersion:            int32(receipts.DefaultReceiptStorage),
		EVMPersistentTxReceiptsMax: receipts.DefaultMaxReceipts,
		SessionDuration:            600,
		EVMAccountsEnabled:         false,
		EVMDebugEnabled:            false,

		Oracle:              "",
		DeployEnabled:       true,
		CallEnabled:         true,
		CallSessionDuration: 1,
		BootLegacyDPoS:       false,
		DPOSVersion:          1,
	}
	cfg.TransferGateway = gateway.DefaultConfig(cfg.RPCProxyPort)
	cfg.LoomCoinTransferGateway = gateway.DefaultLoomCoinTGConfig(cfg.RPCProxyPort)
	cfg.PlasmaCash = plasmacfg.DefaultConfig()
	cfg.AppStore = store.DefaultConfig()
	cfg.HsmConfig = hsmpv.DefaultConfig()
	cfg.TxLimiter = throttle.DefaultTxLimiterConfig()

	cfg.DPOSv2OracleConfig = dposv2OracleCfg.DefaultConfig()
	cfg.CachingStoreConfig = store.DefaultCachingStoreConfig()
	cfg.Metrics = DefaultMetrics()
	cfg.Karma = DefaultKarmaConfig()
	return cfg
}

// Clone returns a deep clone of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c
	clone.TransferGateway = c.TransferGateway.Clone()
	clone.LoomCoinTransferGateway = c.LoomCoinTransferGateway.Clone()
	clone.PlasmaCash = c.PlasmaCash.Clone()
	clone.AppStore = c.AppStore.Clone()
	clone.HsmConfig = c.HsmConfig.Clone()
	clone.TxLimiter = c.TxLimiter.Clone()
	return &clone
}

func (c *Config) fullPath(p string) string {
	full, err := filepath.Abs(path.Join(c.RootDir, p))
	if err != nil {
		panic(err)
	}
	return full
}

func (c *Config) RootPath() string {
	return c.fullPath(c.RootDir)
}

func (c *Config) GenesisPath() string {
	return c.fullPath(c.GenesisFile)
}

func (c *Config) PluginsPath() string {
	return c.fullPath(c.PluginsDir)
}

func (c *Config) QueryServerPort() (int32, error) {
	hostPort := strings.Split(c.QueryServerHost, ":")
	port, err := strconv.ParseInt(hostPort[2], 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(port), nil
}

func (c *Config) WriteToFile(filename string) error {
	var buf bytes.Buffer
	cfgTemplate, err := parseCfgTemplate()
	if err != nil {
		return err
	}
	if err := cfgTemplate.Execute(&buf, c); err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

var cfgTemplate *template.Template

func parseCfgTemplate() (*template.Template, error) {
	if cfgTemplate != nil {
		return cfgTemplate, nil
	}

	var err error
	cfgTemplate, err = template.New("loomYamlTemplate").Parse(defaultLoomYamlTemplate)
	if err != nil {
		return nil, err
	}
	return cfgTemplate, nil
}

const defaultLoomYamlTemplate = `# Loom Node config file
# See https://loomx.io/developers/docs/en/loom-yaml.html for additional info.

# 
# Cluster-wide settings that must not change after cluster is initialized.
#

# Cluster ID
ChainID: "{{ .ChainID }}"
RegistryVersion: {{ .RegistryVersion }}
ReceiptsVersion: {{ .ReceiptsVersion }}
EVMPersistentTxReceiptsMax: {{ .EVMPersistentTxReceiptsMax }}
EVMAccountsEnabled: {{ .EVMAccountsEnabled }}
DPOSVersion: {{ .DPOSVersion }}

CreateEmptyBlocks: {{ .CreateEmptyBlocks }}

#
# Network
#

QueryServerHost: "{{ .QueryServerHost }}"
RPCListenAddress: "{{ .RPCListenAddress }}"
RPCProxyPort: {{ .RPCProxyPort }}
RPCBindAddress: "{{ .RPCBindAddress }}"
EventDispatcherURI: "{{ .EventDispatcherURI }}"
Peers: "{{ .Peers }}"
PersistentPeers: "{{ .PersistentPeers }}"

#
# Karma
#

Oracle: {{ .Oracle }}
DeployEnabled: {{ .DeployEnabled }}
CallEnabled: {{ .CallEnabled }}
SessionDuration: {{ .SessionDuration }}
KarmaEnabled: {{ .KarmaEnabled }}
KarmaMaxCallCount: {{ .KarmaMaxCallCount }}
KarmaSessionDuration: {{ .KarmaSessionDuration }}

#
# Logging
#

LogDestination: "{{ .LogDestination }}"
ContractLogLevel: "{{ .ContractLogLevel }}"
LoomLogLevel: "{{ .LoomLogLevel }}"
BlockchainLogLevel: "{{ .BlockchainLogLevel }}"
LogStateDB: {{ .LogStateDB }}
LogEthDbBatch: {{ .LogEthDbBatch }}

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

#
# Plasma Cash
#

PlasmaCash:
  ContractEnabled: {{ .PlasmaCash.ContractEnabled }}
  OracleEnabled: {{ .PlasmaCash.OracleEnabled }}

# These should pretty much never be changed
RootDir: "{{ .RootDir }}"
DBName: "{{ .DBName }}"
GenesisFile: "{{ .GenesisFile }}"
PluginsDir: "{{ .PluginsDir }}"

#
# Here be dragons, don't change the defaults unless you know what you're doing
#

UseCheckTx: {{ .UseCheckTx }}
EVMDebugEnabled: {{ .EVMDebugEnabled }}
`
