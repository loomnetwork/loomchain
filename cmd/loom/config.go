package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/viper"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/gateway"
	"github.com/loomnetwork/loomchain/plugin"
	receipts "github.com/loomnetwork/loomchain/receipts/factory"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"

	plasmaConfig "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/config"
)

func decodeHexString(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("string has no hex prefix")
	}

	return hex.DecodeString(s[2:])
}

type Config struct {
	RootDir            string
	DBName             string
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
	CreateEmptyBlocks     bool
	SessionMaxAccessCount int64
	SessionDuration       int64
	LogStateDB            bool
	LogEthDbBatch         bool
	UseCheckTx            bool
	RegistryVersion       int32
	ReceiptsVersion       int32
	TransferGateway       *gateway.TransferGatewayConfig
	PlasmaCash            *plasmaConfig.PlasmaCashSerializableConfig
	// When this setting is enabled Loom EVM accounts are hooked up to the builtin ethcoin Go contract,
	// which makes it possible to use the payable/transfer features of the EVM to transfer ETH in
	// Solidity contracts running on the Loom EVM. This setting is disabled by default, which means
	// all the EVM accounts always have a zero balance.
	EVMAccountsEnabled  bool
	EVMDebugEnabled     bool
	EVMPreImagesEnabled bool

	Oracle        string
	DeployEnabled bool
	CallEnabled   bool

	KarmaEnabled         bool
	KarmaMaxCallCount    int64
	KarmaSessionDuration int64
	KarmaMaxDeployCount  int64
}

// Loads loom.yml from ./ or ./config
func parseConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("LOOM")

	v.SetConfigName("loom")                       // name of config file (without extension)
	v.AddConfigPath(".")                          // search root directory
	v.AddConfigPath(filepath.Join(".", "config")) // search root directory /config

	v.ReadInConfig()
	conf := DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
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

func DefaultConfig() *Config {
	cfg := &Config{
		RootDir:             ".",
		DBName:              "app",
		GenesisFile:         "genesis.json",
		PluginsDir:          "contracts",
		QueryServerHost:     "tcp://127.0.0.1:9999",
		RPCListenAddress:    "tcp://0.0.0.0:46657", //TODO this is an ephemeral port in linux, we should move this
		EventDispatcherURI:  "",
		ContractLogLevel:    "info",
		LoomLogLevel:        "info",
		LogDestination:      "",
		BlockchainLogLevel:  "error",
		Peers:               "",
		PersistentPeers:     "",
		ChainID:             "",
		RPCProxyPort:        46658,
		RPCBindAddress:      "tcp://0.0.0.0:46658",
		CreateEmptyBlocks:   true,
		LogStateDB:          false,
		LogEthDbBatch:       false,
		UseCheckTx:          true,
		RegistryVersion:     int32(registry.RegistryV1),
		ReceiptsVersion:     int32(receipts.DefaultReceiptHandlerVersion),
		SessionDuration:     600,
		EVMAccountsEnabled:  false,
		EVMDebugEnabled:     false,
		EVMPreImagesEnabled: false, //TODO hook this up, just adding it for later

		Oracle:        "",
		DeployEnabled: true,
		CallEnabled:   true,

		KarmaEnabled:         false,
		KarmaMaxCallCount:    0,
		KarmaSessionDuration: 0,
		KarmaMaxDeployCount:  0,
	}
	cfg.TransferGateway = gateway.DefaultConfig(cfg.RPCProxyPort)
	cfg.PlasmaCash = plasmaConfig.DefaultConfig()
	return cfg
}

func (c *Config) QueryServerPort() (int32, error) {
	hostPort := strings.Split(c.QueryServerHost, ":")
	port, err := strconv.ParseInt(hostPort[2], 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(port), nil
}

type contractConfig struct {
	VMTypeName string          `json:"vm"`
	Format     string          `json:"format,omitempty"`
	Name       string          `json:"name,omitempty"`
	Location   string          `json:"location"`
	Init       json.RawMessage `json:"init"`
}

func (c contractConfig) VMType() vm.VMType {
	return vm.VMType(vm.VMType_value[c.VMTypeName])
}

type genesis struct {
	Contracts []contractConfig `json:"contracts"`
}

func readGenesis(path string) (*genesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(file)

	var gen genesis
	err = dec.Decode(&gen)
	if err != nil {
		return nil, err
	}

	return &gen, nil
}

func marshalInit(pb proto.Message) (json.RawMessage, error) {
	var buf bytes.Buffer
	marshaler, err := contractpb.MarshalerFactory(plugin.EncodingType_JSON)
	if err != nil {
		return nil, err
	}
	err = marshaler.Marshal(&buf, pb)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(buf.Bytes()), nil
}

func defaultGenesis(cfg *Config, validator *loom.Validator) (*genesis, error) {
	dposInit, err := marshalInit(&dpos.InitRequest{
		Params: &dpos.Params{
			WitnessCount:        21,
			ElectionCycleLength: 604800, // one week
			MinPowerFraction:    5,      // 20%
		},
		Validators: []*loom.Validator{
			validator,
		},
	})
	if err != nil {
		return nil, err
	}

	contracts := []contractConfig{
		contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "coin",
			Location:   "coin:1.0.0",
		},
		contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "dpos",
			Location:   "dpos:1.0.0",
			Init:       dposInit,
		},
	}

	//If this is enabled lets default to giving a genesis file with the plasma_cash contract
	if cfg.PlasmaCash.ContractEnabled == true {
		contracts = append(contracts, contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "plasmacash",
			Location:   "plasmacash:1.0.0",
			//Init:       plasmacashInit,
		})
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts,
			contractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "ethcoin",
				Location:   "ethcoin:1.0.0",
			},
			contractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "addressmapper",
				Location:   "addressmapper:0.1.0",
			},
			contractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "gateway",
				Location:   "gateway:0.1.0",
			})
	}

	if cfg.KarmaEnabled {
		karmaInitRequest := ktypes.KarmaInitRequest{
			Sources: []*ktypes.KarmaSourceReward{
				{Name: "sms", Reward: 1},
				{Name: "oauth", Reward: 3},
				{Name: "token", Reward: 4},
			},
		}
		oracle, err := loom.ParseAddress(cfg.Oracle)
		if err == nil {
			karmaInitRequest.Oracle = oracle.MarshalPB()
		}
		karmaInit, err := marshalInit(&karmaInitRequest)

		if err != nil {
			return nil, err
		}
		contracts = append(contracts, contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "karma",
			Location:   "karma:1.0.0",
			Init:       karmaInit,
		})
	}

	return &genesis{
		Contracts: contracts,
	}, nil
}

type ContractCodeLoader interface {
	LoadContractCode(location string, init json.RawMessage) ([]byte, error)
}

type PluginCodeLoader struct {
}

func (l *PluginCodeLoader) LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
	// just verify that it's json
	body, err := init.MarshalJSON()
	if err != nil {
		return nil, err
	}

	req := &plugin.Request{
		ContentType: plugin.EncodingType_JSON,
		Body:        body,
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	pluginCode := &plugin.PluginCode{
		Name:  location,
		Input: input,
	}
	return proto.Marshal(pluginCode)
}

type TruffleContract struct {
	ByteCodeStr string `json:"bytecode"`
}

func (c *TruffleContract) ByteCode() ([]byte, error) {
	return decodeHexString(c.ByteCodeStr)
}

type TruffleCodeLoader struct {
}

func (l *TruffleCodeLoader) LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	var contract TruffleContract
	enc := json.NewDecoder(file)
	err = enc.Decode(&contract)
	if err != nil {
		return nil, err
	}

	return contract.ByteCode()
}

type SolidityCodeLoader struct {
}

func (l *SolidityCodeLoader) LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	output, err := vm.MarshalSolOutput(file)
	if err != nil {
		return nil, err
	}

	return hex.DecodeString(output.Text)
}

type HexCodeLoader struct {
}

func (l *HexCodeLoader) LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
	b, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, err
	}

	return hex.DecodeString(string(b))
}
