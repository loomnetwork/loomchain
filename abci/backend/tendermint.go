package backend

import (
	"errors"
	"os"
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
	ChainID() (string, error)
	Init() error
	Destroy() error
	Run(app abci.Application, qs *rpc.QueryServer) error
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

func (b *TendermintBackend) Init() error {
	config, err := parseConfig()
	if err != nil {
		return err
	}

	// genesis file
	genFile := config.GenesisFile()
	if util.FileExists(genFile) {
		return errors.New("genesis file already exists")
	}

	// private validator
	privValFile := config.PrivValidatorFile()
	if util.FileExists(privValFile) {
		return errors.New("private validator file already exists")
	}

	privValidator := types.GenPrivValidatorFS(privValFile)
	privValidator.Save()

	genDoc := types.GenesisDoc{
		ChainID: "default",
	}
	genDoc.Validators = []types.GenesisValidator{{
		PubKey: privValidator.GetPubKey(),
		Power:  10,
	}}

	err = genDoc.SaveAs(genFile)
	if err != nil {
		return err
	}

	return nil
}

func (b *TendermintBackend) ChainID() (string, error) {
	config, err := parseConfig()
	if err != nil {
		return "", err
	}

	genDoc, err := types.GenesisDocFromFile(config.GenesisFile())
	if err != nil {
		return "", err
	}

	return genDoc.ChainID, nil
}

func (b *TendermintBackend) Destroy() error {
	config, err := parseConfig()
	if err != nil {
		return err
	}

	os.Remove(config.GenesisFile())
	os.Remove(config.PrivValidatorFile())
	os.RemoveAll(config.DBDir())
	return nil
}

func (b *TendermintBackend) Run(app abci.Application, qs *rpc.QueryServer) error {
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

	qs.Start()

	// Trap signal, run forever.
	n.RunForever()
	return nil
}
