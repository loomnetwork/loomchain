package backend

import (
	"path/filepath"

	"github.com/spf13/viper"
	abci "github.com/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loom/log"
)

type Backend interface {
	Run(app abci.Application) error
}

const (
	homeFlag = "home"
)

type TendermintBackend struct {
}

// ParseConfig retrieves the default environment configuration,
// sets up the Tendermint root and ensures that the root exists
func parseConfig() (*cfg.Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("TM")
	v.SetDefault(homeFlag, "./tendermint")

	homeDir := v.GetString(homeFlag)
	v.Set(homeFlag, homeDir)
	v.SetConfigName("config")                         // name of config file (without extension)
	v.AddConfigPath(homeDir)                          // search root directory
	v.AddConfigPath(filepath.Join(homeDir, "config")) // search root directory /config

	conf := cfg.DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	conf.SetRoot(conf.RootDir)
	cfg.EnsureRoot(conf.RootDir)
	return conf, err
}

func (b *TendermintBackend) Run(app abci.Application) error {
	logger := log.Root
	cfg, err := parseConfig()
	if err != nil {
		return err
	}

	// Create & start tendermint node
	n, err := node.NewNode(cfg,
		types.LoadOrGenPrivValidatorFS(cfg.PrivValidatorFile()),
		proxy.NewLocalClientCreator(app),
		node.DefaultGenesisDocProviderFunc(cfg),
		node.DefaultDBProvider,
		logger.With("module", "node"),
	)
	if err != nil {
		return err
	}

	err = n.Start()
	if err != nil {
		return err
	}

	// Trap signal, run forever.
	n.RunForever()
	return nil
}
