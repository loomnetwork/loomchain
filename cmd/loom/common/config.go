package common

import (
	"fmt"
	"path/filepath"

	"github.com/loomnetwork/loomchain/config"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	"github.com/spf13/viper"
)

// Loads loom.yml from ./ or ./config
func ParseConfig() (*config.Config, error) {
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
	conf.HsmConfig, err = ParseHSMConfig()
	if err != nil {
		fmt.Println("THIS IS ERROR MSG : ", err)
		return nil, err
	}
	return conf, err
}

func ParseHSMConfig() (*hsmpv.HsmConfig, error) {
	v := viper.New()
	v.SetConfigName("loom_hsm")
	v.AddConfigPath(".")                          // search root directory
	v.AddConfigPath(filepath.Join(".", "config")) // search root directory /config
	v.AddConfigPath("./../../../")
	if err := v.ReadInConfig(); err != nil && err != err.(viper.ConfigFileNotFoundError) {
		return nil, err
	}
	cfg := hsmpv.DefaultConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
