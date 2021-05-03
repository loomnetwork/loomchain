package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/loomnetwork/loomchain/auth"
	plasmacfg "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/config"
	genesiscfg "github.com/loomnetwork/loomchain/config/genesis"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/evm"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	receipts "github.com/loomnetwork/loomchain/receipts/handler"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	blockindex "github.com/loomnetwork/loomchain/store/block_index"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/loomnetwork/loomchain/db"

	"github.com/loomnetwork/loomchain/fnConsensus"
)

type (
	Genesis        = genesiscfg.Genesis
	ContractConfig = genesiscfg.ContractConfig
)
type Config struct {
	// Cluster
	ChainID                    string
	RegistryVersion            int32
	ReceiptsVersion            int32
	EVMPersistentTxReceiptsMax uint64

	// When this setting is enabled Loom EVM accounts are hooked up to the builtin ethcoin Go contract,
	// which makes it possible to use the payable/transfer features of the EVM to transfer ETH in
	// Solidity contracts running on the Loom EVM. This setting is disabled by default, which means
	// all the EVM accounts always have a zero balance.
	EVMAccountsEnabled bool
	DPOSVersion        int64

	// Controls whether or not empty blocks should be generated periodically if there are no txs or
	// AppHash changes. Defaults to true.
	CreateEmptyBlocks bool

	// Enable mempool.wal
	MempoolWalEnabled bool

	// Network
	RPCListenAddress     string
	RPCProxyPort         int32
	RPCBindAddress       string
	UnsafeRPCBindAddress string
	UnsafeRPCEnabled     bool

	Peers           string
	PersistentPeers string

	// Throttle
	Oracle                      string
	DeployEnabled               bool
	CallEnabled                 bool
	SessionDuration             int64
	Karma                       *KarmaConfig
	GoContractDeployerWhitelist *throttle.GoContractDeployerWhitelistConfig
	TxLimiter                   *throttle.TxLimiterConfig
	ContractTxLimiter           *throttle.ContractTxLimiterConfig
	// Logging
	LogDestination          string
	ContractLogLevel        string
	LoomLogLevel            string
	BlockchainLogLevel      string
	LogStateDB              bool
	LogEthDbBatch           bool
	Metrics                 *Metrics
	SampleGoContractEnabled bool

	//ChainConfig
	ChainConfig *ChainConfigConfig

	//DeployerWhitelist
	DeployerWhitelist *DeployerWhitelistConfig

	// UserDeployerWhitelist
	UserDeployerWhitelist *UserDeployerWhitelistConfig

	// Transfer gateway
	TransferGateway         *TransferGatewayConfig
	LoomCoinTransferGateway *TransferGatewayConfig
	TronTransferGateway     *TransferGatewayConfig
	BinanceTransferGateway  *TransferGatewayConfig
	BscTransferGateway      *TransferGatewayConfig

	// Plasma Cash
	PlasmaCash *plasmacfg.PlasmaCashSerializableConfig
	// Blockstore config
	BlockStore      *store.BlockStoreConfig
	BlockIndexStore *blockindex.BlockIndexStoreConfig
	// Cashing store
	CachingStoreConfig *store.CachingStoreConfig

	//Prometheus
	PrometheusPushGateway *PrometheusPushGatewayConfig

	//Contracts
	ContractLoaders []string
	//Hsm
	HsmConfig *hsmpv.HsmConfig

	// Oracle serializable
	// todo Cannot be read in from file due to nested pointers to structs.
	DPOSv2OracleConfig *OracleSerializableConfig

	// AppStore
	AppStore *store.AppStoreConfig

	// Should pretty much never be changed
	RootDir     string
	DBName      string
	DBBackend   string
	GenesisFile string
	PluginsDir  string

	DBBackendConfig *DBBackendConfig

	// Event store
	EventStore      *events.EventStoreConfig
	EventDispatcher *events.EventDispatcherConfig

	FnConsensus *FnConsensusConfig

	Auth *auth.Config

	EvmStore *evm.EvmStoreConfig
	// Allow deployment of named EVM contracts (should only be used in tests!)
	AllowNamedEvmContracts bool

	// Dragons
	EVMDebugEnabled bool
	// Set to true to disable minimum required build number check on node startup
	SkipMinBuildCheck bool

	Web3 *eth.Web3Config
	Geth *GethConfig
	DPOS *DPOSConfig
}

type GethConfig struct {
	EnableStateObjectDirtyStorageKeysSorting bool
	EnableTrieDatabasePreimageKeysSorting    bool
}

func DefaultGethConfig() *GethConfig {
	return &GethConfig{
		EnableStateObjectDirtyStorageKeysSorting: false,
		EnableTrieDatabasePreimageKeysSorting:    false,
	}
}

type Metrics struct {
	BlockIndexStore bool
	EventHandling   bool
	Database        bool
}

type FnConsensusConfig struct {
	Enabled bool
	Reactor *fnConsensus.ReactorConfigParsable
}

func DefaultFnConsensusConfig() *FnConsensusConfig {
	return &FnConsensusConfig{
		Enabled: false,
		Reactor: fnConsensus.DefaultReactorConfigParsable(),
	}
}

type DPOSConfig struct {
	BootstrapNodes           []string
	TotalStakedCacheDuration int64
}

func DefaultDPOSConfig() *DPOSConfig {
	return &DPOSConfig{
		BootstrapNodes: []string{
			"default:0x0e99fc16e32e568971908f2ce54b967a42663a26",
			"default:0xac3211caecc45940a6d2ba006ca465a647d8464f",
			"default:0x69c48768dbac492908161be787b7a5658192df35",
			"default:0x2a3a7c850586d4f80a12ac1952f88b1b69ef48e1",
			"default:0x4a1b8b15e50ce63cc6f65603ea79be09206cae70",
			"default:0x0ce7b61c97a6d5083356f115288f9266553e191e",
		},
		TotalStakedCacheDuration: 60, // 60 seconds
	}
}

func (dposCfg *DPOSConfig) BootstrapNodesList() map[string]bool {
	bootstrapNodesList := map[string]bool{}
	for _, addr := range dposCfg.BootstrapNodes {
		bootstrapNodesList[strings.ToLower(addr)] = true
	}
	return bootstrapNodesList
}

type DBBackendConfig struct {
	CacheSizeMegs   int
	WriteBufferMegs int
}

type KarmaConfig struct {
	Enabled         bool  // Activate karma module
	ContractEnabled bool  // Allows you to deploy karma contract to collect data even if chain doesn't use it
	UpkeepEnabled   bool  // Adds an upkeep cost to deployed and active contracts for each user
	MaxCallCount    int64 // Maximum number call transactions per session duration
	SessionDuration int64 // Session length in seconds
}

type PrometheusPushGatewayConfig struct {
	Enabled           bool   //Enable publishing via a Prometheus Pushgatewa
	PushGateWayUrl    string //host:port or ip:port of the Pushgateway
	PushRateInSeconds int64  // Frequency with which to push metrics to Pushgateway
	JobName           string
}

type ChainConfigConfig struct {
	// Allow deployment of the ChainConfig contract
	ContractEnabled bool
	// Allow a validator node to auto-enable features supported by the current build
	AutoEnableFeatures bool
	// How long to wait (in seconds) after the node starts before attempting to auto-enable features
	EnableFeatureStartupDelay int64
	// Frequency (in seconds) with which the node should auto-enable features
	EnableFeatureInterval int64
	// DAppChain URI feature auto-enabler should use to query the chain
	DAppChainReadURI string
	// DAppChain URI feature auto-enabler should use to submit txs to the chain
	DAppChainWriteURI string
	// Log level for feature auto-enabler
	LogLevel string
	// Log destination for feature auto-enabler
	LogDestination string
}

type DeployerWhitelistConfig struct {
	ContractEnabled bool
}

type UserDeployerWhitelistConfig struct {
	ContractEnabled bool
}

func DefaultDBBackendConfig() *DBBackendConfig {
	return &DBBackendConfig{
		CacheSizeMegs:   1042, //1 Gigabyte
		WriteBufferMegs: 500,  //500 megabyte
	}
}

func DefaultMetrics() *Metrics {
	return &Metrics{
		BlockIndexStore: false,
		EventHandling:   true,
		Database:        true,
	}
}

func DefaultKarmaConfig() *KarmaConfig {
	return &KarmaConfig{
		Enabled:         false,
		ContractEnabled: false,
		UpkeepEnabled:   false,
		MaxCallCount:    0,
		SessionDuration: 0,
	}
}

func DefaultPrometheusPushGatewayConfig() *PrometheusPushGatewayConfig {
	return &PrometheusPushGatewayConfig{
		Enabled:           false,
		PushGateWayUrl:    "http://localhost:9091",
		PushRateInSeconds: 60,
		JobName:           "Loommetrics",
	}
}

func DefaultChainConfigConfig(rpcProxyPort int32) *ChainConfigConfig {
	return &ChainConfigConfig{
		ContractEnabled:           true,
		AutoEnableFeatures:        true,
		EnableFeatureStartupDelay: 5 * 60,  // wait 5 mins after startup before auto-enabling features
		EnableFeatureInterval:     15 * 60, // auto-enable features every 15 minutes
		DAppChainReadURI:          fmt.Sprintf("http://127.0.0.1:%d/query", rpcProxyPort),
		DAppChainWriteURI:         fmt.Sprintf("http://127.0.0.1:%d/rpc", rpcProxyPort),
		LogLevel:                  "info",
		LogDestination:            "file://chainconfig.log",
	}
}

func DefaultDeployerWhitelistConfig() *DeployerWhitelistConfig {
	return &DeployerWhitelistConfig{
		ContractEnabled: false,
	}
}

func DefaultUserDeployerWhitelistConfig() *UserDeployerWhitelistConfig {
	return &UserDeployerWhitelistConfig{
		ContractEnabled: true,
	}
}

//Structure for LOOM ENV

type Env struct {
	Version         string `json:"version"`
	Build           string `json:"build"`
	BuildVariant    string `json:"buildvariant"`
	GitSha          string `json:"gitsha"`
	GoLoom          string `json:"goloom"`
	TransferGateway string `json:"transfergateway"`
	GoEthereum      string `json:"goethereum"`
	GoPlugin        string `json:"goplugin"`
	Btcd            string `json:"btcd"`
	PluginPath      string `json:"pluginpath"`
	Peers           string `json:"peers"`
}

// TODO: Move to loomchain/rpc package
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

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	conf := DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	return conf, err
}

func ParseConfigFrom(filename string) (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("LOOM")

	v.SetConfigName(filename)                      // name of config file (without extension)
	v.AddConfigPath("./")                          // search root directory
	v.AddConfigPath(filepath.Join("./", "config")) // search root directory /config
	v.AddConfigPath("./../../../")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
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
	if err := dec.Decode(&gen); err != nil {
		return nil, errors.Wrap(err, "failed to decode loom genesis file")
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
		RPCListenAddress:           "tcp://127.0.0.1:46657", // TODO this is an ephemeral port in linux, we should move this
		ContractLogLevel:           "info",
		LoomLogLevel:               "info",
		LogDestination:             "",
		BlockchainLogLevel:         "error",
		Peers:                      "",
		PersistentPeers:            "",
		ChainID:                    "",
		RPCProxyPort:               46658,
		RPCBindAddress:             "tcp://0.0.0.0:46658",
		UnsafeRPCEnabled:           false,
		UnsafeRPCBindAddress:       "tcp://127.0.0.1:26680",
		CreateEmptyBlocks:          true,
		MempoolWalEnabled:          false,
		ContractLoaders:            []string{"static"},
		LogStateDB:                 false,
		LogEthDbBatch:              false,
		RegistryVersion:            int32(registry.RegistryV2),
		ReceiptsVersion:            int32(receipts.ReceiptHandlerLevelDb),
		EVMPersistentTxReceiptsMax: receipts.DefaultMaxReceipts,
		SessionDuration:            600,
		EVMAccountsEnabled:         false,
		EVMDebugEnabled:            false,
		SampleGoContractEnabled:    false,

		Oracle:                 "",
		DeployEnabled:          true,
		CallEnabled:            true,
		DPOSVersion:            3,
		AllowNamedEvmContracts: false,
		SkipMinBuildCheck:      false,
	}
	cfg.TransferGateway = DefaultTGConfig(cfg.RPCProxyPort)
	cfg.LoomCoinTransferGateway = DefaultLoomCoinTGConfig(cfg.RPCProxyPort)
	cfg.TronTransferGateway = DefaultTronTGConfig(cfg.RPCProxyPort)
	cfg.BinanceTransferGateway = DefaultBinanceTGConfig()
	cfg.BscTransferGateway = DefaultBscLoomCoinTGConfig(cfg.RPCProxyPort)
	cfg.PlasmaCash = plasmacfg.DefaultConfig()
	cfg.AppStore = store.DefaultConfig()
	cfg.HsmConfig = hsmpv.DefaultConfig()
	cfg.TxLimiter = throttle.DefaultTxLimiterConfig()
	cfg.ContractTxLimiter = throttle.DefaultContractTxLimiterConfig()
	cfg.GoContractDeployerWhitelist = throttle.DefaultGoContractDeployerWhitelistConfig()
	cfg.DPOSv2OracleConfig = DefaultDPOS2OracleConfig()
	cfg.CachingStoreConfig = store.DefaultCachingStoreConfig()
	cfg.BlockStore = store.DefaultBlockStoreConfig()
	cfg.BlockIndexStore = blockindex.DefaultBlockIndexStoreConfig()
	cfg.Metrics = DefaultMetrics()
	cfg.Karma = DefaultKarmaConfig()
	cfg.ChainConfig = DefaultChainConfigConfig(cfg.RPCProxyPort)
	cfg.DeployerWhitelist = DefaultDeployerWhitelistConfig()
	cfg.UserDeployerWhitelist = DefaultUserDeployerWhitelistConfig()
	cfg.DBBackendConfig = DefaultDBBackendConfig()
	cfg.PrometheusPushGateway = DefaultPrometheusPushGatewayConfig()
	cfg.EventDispatcher = events.DefaultEventDispatcherConfig()
	cfg.EventStore = events.DefaultEventStoreConfig()
	cfg.EvmStore = evm.DefaultEvmStoreConfig()
	cfg.Web3 = eth.DefaultWeb3Config()
	cfg.Geth = DefaultGethConfig()
	cfg.DPOS = DefaultDPOSConfig()

	cfg.FnConsensus = DefaultFnConsensusConfig()

	cfg.Auth = auth.DefaultConfig()
	return cfg
}

func (c *Config) AddressMapperContractEnabled() bool {
	return c.TransferGateway.ContractEnabled ||
		c.LoomCoinTransferGateway.ContractEnabled ||
		c.TronTransferGateway.ContractEnabled ||
		c.PlasmaCash.ContractEnabled ||
		c.Auth.AddressMapperContractRequired()
}

// Clone returns a deep clone of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c
	clone.TransferGateway = c.TransferGateway.Clone()
	clone.LoomCoinTransferGateway = c.LoomCoinTransferGateway.Clone()
	clone.TronTransferGateway = c.TronTransferGateway.Clone()
	clone.PlasmaCash = c.PlasmaCash.Clone()
	clone.AppStore = c.AppStore.Clone()
	clone.HsmConfig = c.HsmConfig.Clone()
	clone.TxLimiter = c.TxLimiter.Clone()
	clone.ContractTxLimiter = c.ContractTxLimiter.Clone()
	clone.EventStore = c.EventStore.Clone()
	clone.EventDispatcher = c.EventDispatcher.Clone()
	clone.Auth = c.Auth.Clone()
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
MempoolWalEnabled: {{ .MempoolWalEnabled }}
#
# Network
#
RPCListenAddress: "{{ .RPCListenAddress }}"
RPCProxyPort: {{ .RPCProxyPort }}
RPCBindAddress: "{{ .RPCBindAddress }}"
UnsafeRPCEnabled: {{ .UnsafeRPCEnabled }}
UnsafeRPCBindAddress: "{{ .UnsafeRPCBindAddress }}"
Peers: "{{ .Peers }}"
PersistentPeers: "{{ .PersistentPeers }}"
#
# Throttle
#
Oracle: "{{ .Oracle }}"
DeployEnabled: {{ .DeployEnabled }}
CallEnabled: {{ .CallEnabled }}
SessionDuration: {{ .SessionDuration }}
Karma:
  Enabled: {{ .Karma.Enabled }}
  ContractEnabled: {{ .Karma.ContractEnabled }}
  UpkeepEnabled: {{ .Karma.UpkeepEnabled }}
  MaxCallCount: {{ .Karma.MaxCallCount }}
  SessionDuration: {{ .Karma.SessionDuration }}
GoContractDeployerWhitelist:
  Enabled: {{ .GoContractDeployerWhitelist.Enabled }}
  DeployerAddressList:
  {{- range .GoContractDeployerWhitelist.DeployerAddressList}}
    - "{{. -}}"
  {{- end}}
TxLimiter:
  Enabled: {{ .TxLimiter.Enabled }}
  SessionDuration: {{ .TxLimiter.SessionDuration }}
  MaxTxsPerSession: {{ .TxLimiter.MaxTxsPerSession }} 
ContractTxLimiter:
  Enabled: {{ .ContractTxLimiter.Enabled }}
  ContractDataRefreshInterval: {{ .ContractTxLimiter.ContractDataRefreshInterval }}
  TierDataRefreshInterval: {{ .ContractTxLimiter.TierDataRefreshInterval }}

#
# ContractLoader
#
ContractLoaders:
  {{- range .ContractLoaders}}
  - "{{. -}}" 
  {{- end}}
#
# Logging
#
LogDestination: "{{ .LogDestination }}"
ContractLogLevel: "{{ .ContractLogLevel }}"
LoomLogLevel: "{{ .LoomLogLevel }}"
BlockchainLogLevel: "{{ .BlockchainLogLevel }}"
LogStateDB: {{ .LogStateDB }}
LogEthDbBatch: {{ .LogEthDbBatch }}
Metrics:
  BlockIndexStore: {{ .Metrics.BlockIndexStore }} 
  EventHandling: {{ .Metrics.EventHandling }}
  Database: {{ .Metrics.Database }}

#
# ChainConfig
#
ChainConfig:
  # Allow deployment of the ChainConfig contract
  ContractEnabled: {{ .ChainConfig.ContractEnabled }}
  # Allow a validator node to auto-enable features supported by the current build
  AutoEnableFeatures: {{ .ChainConfig.AutoEnableFeatures }}
  # How long to wait (in seconds) after the node starts before attempting to auto-enable features
  EnableFeatureStartupDelay: {{ .ChainConfig.EnableFeatureStartupDelay }}
  # Frequency (in seconds) with which the node should auto-enable features
  EnableFeatureInterval: {{ .ChainConfig.EnableFeatureInterval }}
  DAppChainReadURI: {{ .ChainConfig.DAppChainReadURI }}
  DAppChainWriteURI: {{ .ChainConfig.DAppChainWriteURI }}
  # Log level for feature auto-enabler
  LogLevel: {{ .ChainConfig.LogLevel }}
  # Log destination for feature auto-enabler
  LogDestination: {{ .ChainConfig.LogDestination }}

#
# DeployerWhitelist
#
DeployerWhitelist:
  ContractEnabled: {{ .DeployerWhitelist.ContractEnabled }}

#
# UserDeployerWhitelist
#
UserDeployerWhitelist:
  ContractEnabled: {{ .UserDeployerWhitelist.ContractEnabled }}
#
# SampleGoContractEnabled
#
SampleGoContractEnabled: {{ .SampleGoContractEnabled }}

#
# Plasma Cash
#
PlasmaCash:
  ContractEnabled: {{ .PlasmaCash.ContractEnabled }}
  OracleEnabled: {{ .PlasmaCash.OracleEnabled }}
#
# Block store
#
BlockStore:
  # None | LRU | 2Q
  CacheAlgorithm: {{ .BlockStore.CacheAlgorithm }}
  CacheSize: {{ .BlockStore.CacheSize }}
BlockIndexStore:  
  Enabled: {{ .BlockIndexStore.Enabled }}
  # goleveldb | cleveldb | memdb
  DBBackend: {{ .BlockIndexStore.DBBackend }}
  DBName: {{ .BlockIndexStore.DBName }}
  CacheSizeMegs: {{ .BlockIndexStore.CacheSizeMegs }}
  WriteBufferMegs: {{ .BlockIndexStore.WriteBufferMegs }}
#
# Cashing store 
#
CachingStoreConfig: 
  CachingEnabled: {{ .CachingStoreConfig.CachingEnabled }}
  # Number of cache shards, value must be a power of two
  Shards: {{ .CachingStoreConfig.Shards }} 
  # Time after we need to evict the key
  EvictionTimeInSeconds: {{ .CachingStoreConfig.EvictionTimeInSeconds }} 
  # interval at which clean up of expired keys will occur
  CleaningIntervalInSeconds: {{ .CachingStoreConfig.CleaningIntervalInSeconds }} 
  # Total size of cache would be: MaxKeys*MaxSizeOfValueInBytes
  MaxKeys: {{ .CachingStoreConfig.MaxKeys }} 
  MaxSizeOfValueInBytes: {{ .CachingStoreConfig.MaxSizeOfValueInBytes }} 
  # Logs operations
  Verbose: {{ .CachingStoreConfig.Verbose }} 
  LogLevel: "{{ .CachingStoreConfig.LogLevel }}" 
  LogDestination: "{{ .CachingStoreConfig.LogDestination }}" 
#
# Prometheus Push Gateway
#
PrometheusPushGateway:
  #Enable publishing via a Prometheus Pushgateway
  Enabled: {{ .PrometheusPushGateway.Enabled }}  
  #host:port or ip:port of the Pushgateway
  PushGateWayUrl: "{{ .PrometheusPushGateway.PushGateWayUrl}}" 
  #Frequency with which to push metrics to Pushgateway
  PushRateInSeconds: {{ .PrometheusPushGateway.PushRateInSeconds}} 
  JobName: "{{ .PrometheusPushGateway.JobName }}"


#
# Hsm 
#
HsmConfig:
  # flag to enable HSM
  HsmEnabled: {{ .HsmConfig.HsmEnabled }}
  # device type of HSM
  HsmDevType: "{{ .HsmConfig.HsmDevType }}"
  # the path of PKCS#11 library
  HsmP11LibPath: "{{ .HsmConfig.HsmP11LibPath }}"
  # connection URL to YubiHSM
  HsmConnURL: {{ .HsmConfig.HsmConnURL }}
  # Auth key ID for YubiHSM
  HsmAuthKeyID: {{ .HsmConfig.HsmAuthKeyID }}
  # Auth password
  HsmAuthPassword: "{{ .HsmConfig.HsmAuthPassword }}"
  # Sign Key ID for YubiHSM
  HsmSignKeyID: {{ .HsmConfig.HsmSignKeyID }}
  # key domain
  HsmSignKeyDomain: {{ .HsmConfig.HsmSignKeyDomain }}

#
# App store
#
AppStore:
  # 1 - IAVL, 2 - MultiReaderIAVL, 3 - MultiWriterAppStore, defaults to 3
  # WARNING: Once a node is initialized with a specific version it can't be switched to another
  #          version without rebuilding the node.
  Version: {{ .AppStore.Version }}
  # If true the app store will be compacted before it's loaded to reclaim disk space.
  CompactOnLoad: {{ .AppStore.CompactOnLoad }}
  # Maximum number of app store versions to keep, if zero old versions will never be deleted.
  MaxVersions: {{ .AppStore.MaxVersions }}
  # Number of seconds to wait after pruning a batch of old versions from the app store.
  # If this is set to zero the app store will only be pruned after a new version is saved.
  PruneInterval: {{ .AppStore.PruneInterval }}
  # Number of versions to prune at a time.
  PruneBatchSize: {{ .AppStore.PruneBatchSize }}
  # If true the app store will write EVM state to both IAVLStore and EvmStore
  # This config works with AppStore Version 3 (MultiWriterAppStore) only
  SaveEVMStateToIAVL: {{ .AppStore.SaveEVMStateToIAVL }}
{{if .EventStore -}}
#
# EventStore
#
EventStore:
  DBName: {{.EventStore.DBName}}
  DBBackend: {{.EventStore.DBBackend}}
{{end}}

{{if .EvmStore -}}
#
# EvmStore
#
EvmStore:
  # DBName defines evm database file name
  DBName: {{.EvmStore.DBName}}
  # DBBackend defines backend EVM store type
  # available backend types are 'goleveldb', or 'cleveldb'
  DBBackend: {{.EvmStore.DBBackend}}
  # CacheSizeMegs defines cache size (in megabytes) of EVM store
  CacheSizeMegs: {{.EvmStore.CacheSizeMegs}}
  # NumCachedRoots defines a number of in-memory cached EVM roots
  NumCachedRoots: {{.EvmStore.NumCachedRoots}}
{{end}}

{{if .Web3 -}}
#
# Configuration of Web3 JSON-RPC methods served on the /eth endpoint.
#
Web3:
  # Specifies the maximum number of blocks eth_getLogs will query per request
  GetLogsMaxBlockRange: {{.Web3.GetLogsMaxBlockRange}}
{{end}}

# 
# FnConsensus reactor on/off switch + config
#
{{- if .FnConsensus }}
FnConsensus:
  Enabled: {{ .FnConsensus.Enabled }}
  {{- if .FnConsensus.Reactor }}
  Reactor:
    # Set to false to make the node forward messages without tracking consensus state
    IsValidator: {{ .FnConsensus.Reactor.IsValidator }}
    FnVoteSigningThreshold: {{ .FnConsensus.Reactor.FnVoteSigningThreshold }}
    {{- if .FnConsensus.Reactor.OverrideValidators }}
    OverrideValidators:
      {{- range $i, $v := .FnConsensus.Reactor.OverrideValidators }}
      - Address: {{ $v.Address }}
        VotingPower: {{ $v.VotingPower }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}

#
# EventDispatcher
#
EventDispatcher:
  # Available dispatcher: "db_indexer" | "log" | "redis"
  Dispatcher: {{.EventDispatcher.Dispatcher}}
  {{if eq .EventDispatcher.Dispatcher "redis"}}
  # Redis will be use when Dispatcher is "redis"
  Redis:
    URI: "{{.EventDispatcher.Redis.URI}}"
  {{end}}
#
# Tx signing & accounts
#
Auth:
  Chains:
    {{- range $k, $v := .Auth.Chains}}
    {{$k}}:
      TxType: "{{.TxType -}}"
      AccountType: {{.AccountType -}}
    {{- end}}
# These should pretty much never be changed
RootDir: "{{ .RootDir }}"
DBName: "{{ .DBName }}"
GenesisFile: "{{ .GenesisFile }}"
PluginsDir: "{{ .PluginsDir }}"

{{if .DPOS -}}
#
# Configuration of DPOSv3 JSON-RPC methods served on /query endpoint.
#
DPOS:
  # Specifies addresses of bootstrap nodes
  BootstrapNodes:
  {{- range .DPOS.BootstrapNodes}}
    - "{{. -}}"
  {{- end}}
  # How long (in seconds) the response from the dpos_total_staked RPC method should be cached.
  TotalStakedCacheDuration: {{ .DPOS.TotalStakedCacheDuration }}
{{end}}

#
# Here be dragons, don't change the defaults unless you know what you're doing
#
EVMDebugEnabled: {{ .EVMDebugEnabled }}
AllowNamedEvmContracts: {{ .AllowNamedEvmContracts }}
# Set to true to disable minimum required build number check on node startup
SkipMinBuildCheck: {{ .SkipMinBuildCheck }}

{{if .Geth -}}
#
# Internal EVM integration settings
#
Geth:
  EnableStateObjectDirtyStorageKeysSorting: {{.Geth.EnableStateObjectDirtyStorageKeysSorting}}
  EnableTrieDatabasePreimageKeysSorting: {{.Geth.EnableTrieDatabasePreimageKeysSorting}}
{{end}}
` + transferGatewayLoomYamlTemplate
