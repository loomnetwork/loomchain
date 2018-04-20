package backend

import (
	"errors"
	"os"

	"github.com/spf13/viper"
	abci "github.com/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/util"
)

type Backend interface {
	ChainID() (string, error)
	Init() error
	Reset(height uint64) error
	Destroy() error
	Start(app abci.Application) error
	RunForever()
}

type TendermintBackend struct {
	RootPath string
	node     *node.Node
}

// ParseConfig retrieves the default environment configuration,
// sets up the Tendermint root and ensures that the root exists
func (b *TendermintBackend) parseConfig() (*cfg.Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("TM")
	v.SetConfigName("config")   // name of config file (without extension)
	v.AddConfigPath(b.RootPath) // search root directory

	conf := cfg.DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	conf.SetRoot(b.RootPath)
	cfg.EnsureRoot(b.RootPath)
	return conf, err
}

func (b *TendermintBackend) Init() error {
	config, err := b.parseConfig()
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

func (b *TendermintBackend) Reset(height uint64) error {
	if height != 0 {
		return errors.New("can only reset back to height 0")
	}
	cfg, err := b.parseConfig()
	if err != nil {
		return err
	}
	return util.IgnoreErrNotExists(os.RemoveAll(cfg.DBDir()))
}

func (b *TendermintBackend) ChainID() (string, error) {
	config, err := b.parseConfig()
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
	config, err := b.parseConfig()
	if err != nil {
		return err
	}

	err = util.IgnoreErrNotExists(os.Remove(config.GenesisFile()))
	if err != nil {
		return err
	}
	err = util.IgnoreErrNotExists(os.Remove(config.PrivValidatorFile()))
	if err != nil {
		return err
	}

	return b.Reset(0)
}

func (b *TendermintBackend) Start(app abci.Application) error {
	logger := log.Root
	cfg, err := b.parseConfig()
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
	b.node = n
	return nil
}

func (b *TendermintBackend) RunForever() {
	// Trap signal, run forever.
	b.node.RunForever()
}
