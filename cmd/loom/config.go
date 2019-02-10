package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/pkg/errors"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/viper"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/vm"
)

func decodeHexString(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("string has no hex prefix")
	}

	return hex.DecodeString(s[2:])
}

// Loads loom.yml from ./ or ./config
func parseConfig() (*config.Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("LOOM")

	v.SetConfigName("loom")                       // name of config file (without extension)
	v.AddConfigPath(".")                          // search root directory
	v.AddConfigPath(filepath.Join(".", "config")) // search root directory /config
	v.AddConfigPath("./../../../")

	v.ReadInConfig()
	conf := config.DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
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

func defaultGenesis(cfg *config.Config, validator *loom.Validator) (*config.Genesis, error) {
	contracts := []config.ContractConfig{
		{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "coin",
			Location:   "coin:1.0.0",
		},
	}

	if cfg.DPOSVersion != 2 {
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

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "dpos",
			Location:   "dpos:1.0.0",
			Init:       dposInit,
		})
	} else {
		dposV2Init, err := marshalInit(&dposv2.InitRequest{
			Params: &dposv2.Params{
				ValidatorCount:      21,
				ElectionCycleLength: 604800, // one week
			},
			Validators: []*loom.Validator{
				validator,
			},
		})
		if err != nil {
			return nil, err
		}

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "dposV2",
			Location:   "dposV2:2.0.0",
			Init:       dposV2Init,
		})
	}

	//If this is enabled lets default to giving a genesis file with the plasma_cash contract
	if cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "plasmacash",
				Location:   "plasmacash:1.0.0",
			})
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "ethcoin",
				Location:   "ethcoin:1.0.0",
			})
	}

	if cfg.TransferGateway.ContractEnabled || cfg.LoomCoinTransferGateway.ContractEnabled || cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "addressmapper",
				Location:   "addressmapper:0.1.0",
			})
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "gateway",
				Location:   "gateway:0.1.0",
			})
	}

	if cfg.LoomCoinTransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "loomcoin-gateway",
				Location:   "loomcoin-gateway:0.1.0",
			})
	}

	if cfg.Karma.Enabled {
		karmaInitRequest := ktypes.KarmaInitRequest{
			Sources: []*ktypes.KarmaSourceReward{
				{Name: "example-award-token", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY,},
			},
			Upkeep: &ktypes.KarmaUpkeepParams{
				Cost:   1,
				Period: 3600,
			},
			Config: &ktypes.KarmaConfig{ MinKarmaToDeploy: karma.DefaultUpkeepCost },
		}
		oracle, err := loom.ParseAddress(cfg.Oracle)
		if err == nil {
			karmaInitRequest.Oracle = oracle.MarshalPB()
		}
		karmaInit, err := marshalInit(&karmaInitRequest)

		if err != nil {
			return nil, err
		}
		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "karma",
			Location:   "karma:1.0.0",
			Init:       karmaInit,
		})
	}

	return &config.Genesis{
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
