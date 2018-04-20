package main

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/viper"

	"github.com/loomnetwork/loom/plugin"
	"github.com/loomnetwork/loom/vm"
)

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
	ByteCode []byte `json:"bytecode"`
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

	return contract.ByteCode, nil
}
