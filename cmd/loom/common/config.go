package common

import (
	"log"
	"path/filepath"

	"github.com/loomnetwork/loomchain/config"
	"github.com/spf13/viper"
)

// Loads loom.yml from ./ or ./config
func ParseConfig() (*config.Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("LOOM")
	PasreConfigFileName("loom", v)
	PasreConfigFileName("loom_hsm", v) // load hsm config if loom_hsm.yaml  exist

	conf := config.DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}

func PasreConfigFileName(name string, v *viper.Viper) {

	v.SetConfigName(name)                         // name of config file (without extension)
	v.AddConfigPath(".")                          // search root directory
	v.AddConfigPath(filepath.Join(".", "config")) // search root directory /config
	v.AddConfigPath("./../../../")
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
}
