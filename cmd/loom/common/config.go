package common

import (
	"path/filepath"

	"github.com/loomnetwork/loomchain/config"
	"github.com/spf13/viper"
)

// FIXME: this is just a copy of parseConfig() from loom.go, need to clean this up so there's only
//        copy

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
	return conf, err
}
