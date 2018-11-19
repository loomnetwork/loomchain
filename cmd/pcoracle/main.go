// +build evm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/config"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/oracle"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

type LoomConfig struct {
	ChainID    string
	PlasmaCash *config.PlasmaCashSerializableConfig
}

func main() {
	loomCfg, err := parseConfig(nil)
	if err != nil {
		panic(errors.Wrapf(err, "unable to parse loom configuration"))
	}

	plasmaCashConfig, err := config.LoadSerializableConfig(loomCfg.ChainID, loomCfg.PlasmaCash)
	if err != nil {
		panic(errors.Wrapf(err, "unable to load plasma cash configuration"))
	}

	// Graceful exit as this is not an error
	if !plasmaCashConfig.OracleEnabled {
		fmt.Println("OracleEnabled flag is false, exiting..")
		return
	}

	oracle := oracle.NewOracle(plasmaCashConfig.OracleConfig)

	if err := oracle.Init(); err != nil {
		panic(errors.Wrapf(err, "unable to init oracle"))
	}

	go oracle.Run()

	http.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(oracle.Status())
	})

	http.Handle("/metrics", promhttp.Handler())

	err = http.ListenAndServe(plasmaCashConfig.StatusServiceAddress, nil)
	if err != nil {
		panic(errors.Wrapf(err, "unable to start status service"))
	}
}

func parseConfig(overrideCfgDirs []string) (*LoomConfig, error) {
	v := viper.New()
	v.SetConfigName("loom")

	if len(overrideCfgDirs) == 0 {
		// look for the loom config file in all the places loom itself does
		v.AddConfigPath(".")
		v.AddConfigPath(filepath.Join(".", "config"))
	} else {
		for _, dir := range overrideCfgDirs {
			v.AddConfigPath(dir)
		}
	}
	v.ReadInConfig()
	conf := &LoomConfig{
		ChainID:    "default",
		PlasmaCash: config.DefaultConfig(),
	}
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}
