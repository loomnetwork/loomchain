package common

import (
	"path/filepath"

	"github.com/loomnetwork/loomchain/config"
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

	v.SetConfigName("loom_hsm")
	v.ReadInConfig()
	err = v.UnmarshalKey("HsmConfig", conf.HsmConfig)
	if err != nil {
		return nil, err
	}
	return conf, err
}
