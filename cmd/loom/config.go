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

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/viper"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
)

func decodeHexString(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("string has no hex prefix")
	}

	return hex.DecodeString(s[2:])
}

type Config struct {
	RootDir               string
	DBName                string
	GenesisFile           string
	PluginsDir            string
	QueryServerHost       string
	EventDispatcherURI    string
	ContractLogLevel      string
	LogDestination        string
	LoomLogLevel          string
	BlockchainLogLevel    string
	Peers                 string
	PersistentPeers       string
	RPCProxyPort          int32
	SessionMaxAccessCount int64
	SessionDuration       int64
	PlasmaCashEnabled     bool
	// Enables the Transfer Gateway Go contract on the node, must be the same on all nodes.
	GatewayContractEnabled bool
	// Enables the Transfer Gateway Oracle, can only be enabled on validators.
	// If this is enabled GatewayContractEnabled must be set to true.
	GatewayOracleEnabled bool
	// URI of Ethereum node that will be used by oracles to listen to Ethereum events.
	EthereumURI string
	// Hex address of Transfer Gateway Solidity contract on Ethereum mainnet
	// e.g. 0x3599a0abda08069e8e66544a2860e628c5dc1190
	GatewayEthAddress string
	MutableOracle	bool
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
	return &Config{
		RootDir:                ".",
		DBName:                 "app",
		GenesisFile:            "genesis.json",
		PluginsDir:             "contracts",
		QueryServerHost:        "tcp://127.0.0.1:9999",
		EventDispatcherURI:     "",
		ContractLogLevel:       "info",
		LoomLogLevel:           "info",
		LogDestination:         "",
		BlockchainLogLevel:     "error",
		Peers:                  "",
		PersistentPeers:        "",
		RPCProxyPort:           46658,
		SessionMaxAccessCount:  0, //Zero is unlimited and disables throttling
		SessionDuration:        600,
		PlasmaCashEnabled:      false,
		GatewayContractEnabled: false,
		GatewayOracleEnabled:   false,
		EthereumURI:            "ws://127.0.0.1:8545",
		GatewayEthAddress:      "",
		MutableOracle:			false,
	}
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
	karmaInit, err := marshalInit(&karma.InitRequest{
		Params: &karma.Params{
			MaxKarma: 10000,
			MutableOracle: cfg.MutableOracle,
			OraclePublicAddress: "irn38gFRpNOzoySXECh5JZVoPm1Hw6UAqCdeqv4IQlM=", // change to real oracle key
			Sources: map[string]int64{
				"sms": 1, //default sources and values
				"oauth": 3, //default sources and values
				"token": 5, //default sources and values
			},
			Validators: []*loom.Validator{
				validator,
			},
		},
	})

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
			Name:       "karma",
			Location:   "karma:1.0.0",
			Init:       karmaInit,
		},
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
	if cfg.PlasmaCashEnabled == true {
		contracts = append(contracts, contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "plasmacash",
			Location:   "plasmacash:1.0.0",
			//Init:       plasmacashInit,
		})
	}

	if cfg.GatewayContractEnabled {
		contracts = append(contracts, contractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "gateway",
			Location:   "gateway:0.1.0",
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