package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/viper"

	"github.com/loomnetwork/loom/plugin"
	"github.com/loomnetwork/loom/vm"
)

func decodeHexString(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("string has no hex prefix")
	}

	return hex.DecodeString(s[2:])
}

type Config struct {
	RootDir         string
	DBName          string
	GenesisFile     string
	PluginsDir      string
	QueryServerHost string
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
		RootDir:         ".",
		DBName:          "app",
		GenesisFile:     "genesis.json",
		PluginsDir:      "contracts",
		QueryServerHost: "tcp://127.0.0.1:9999",
	}
}

type contractConfig struct {
	VMTypeName string          `json:"vm"`
	Format     string          `json:"format,omitempty"`
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
		ContentType: plugin.ContentType_JSON,
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
