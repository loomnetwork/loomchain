// +build evm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/loomnetwork/loomchain/gateway"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LoomConfig struct {
	ChainID                 string
	RPCProxyPort            int32
	LoomCoinTransferGateway *gateway.TransferGatewayConfig
}

func newRootCommand() *cobra.Command {
	var ethGatewayEventSourceTxsPath string
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig(nil)
			if err != nil {
				panic(err)
			}

			orc, err := gateway.CreateLoomCoinOracle(
				cfg.LoomCoinTransferGateway, cfg.ChainID, ethGatewayEventSourceTxsPath,
			)
			if err != nil {
				panic(err)
			}

			go orc.RunWithRecovery()

			http.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(orc.Status())
			})

			http.Handle("/metrics", promhttp.Handler())

			log.Fatal(http.ListenAndServe(cfg.LoomCoinTransferGateway.OracleQueryAddress, nil))
			return nil
		},
	}
	cmd.Flags().StringVar(
		&ethGatewayEventSourceTxsPath, "eth-gateway-event-source-txs", "",
		"JSON file with hashes of Ethereum txs that emitted Gateway events, Oracle will submit the events to the DAppChain")
	return cmd
}

func main() {
	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Loads loom.yml or equivalent from one of the usual location, or if overrideCfgDirs is provided
// from one of those config directories.
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
		ChainID:                 "default",
		RPCProxyPort:            46658,
		LoomCoinTransferGateway: gateway.DefaultLoomCoinTGConfig(46658),
	}
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}
