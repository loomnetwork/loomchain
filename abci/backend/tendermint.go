package backend

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	abci "github.com/tendermint/abci/types"
	crypto "github.com/tendermint/go-crypto"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
	//	pv "github.com/tendermint/tendermint/types/priv_validator"
	pv "github.com/tendermint/tendermint/privval"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
)

type Backend interface {
	ChainID() (string, error)
	Init() (*loom.Validator, error)
	Reset(height uint64) error
	Destroy() error
	Start(app abci.Application) error
	RunForever()
	NodeKey() (string, error)
	EventBus() *types.EventBus
}

type TendermintBackend struct {
	RootPath    string
	node        *node.Node
	OverrideCfg *OverrideConfig
}

func resetPrivValidator(privVal *pv.FilePV, height int64) {
	privVal.LastHeight = height
	privVal.LastRound = 0
	privVal.LastStep = 0
}

// ParseConfig retrieves the default environment configuration,
// sets up the Tendermint root and ensures that the root exists
func (b *TendermintBackend) parseConfig() (*cfg.Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetEnvPrefix("TM")
	v.SetConfigName("config")               // name of config file (without extension)
	v.AddConfigPath(b.RootPath + "/config") // search root directory
	v.ReadInConfig()
	conf := cfg.DefaultConfig()
	err := v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	conf.SetRoot(b.RootPath)
	cfg.EnsureRoot(b.RootPath)
	return conf, err
}

type OverrideConfig struct {
	LogLevel        string
	Peers           string
	PersistentPeers string
}

func (b *TendermintBackend) Init() (*loom.Validator, error) {
	config, err := b.parseConfig()
	if err != nil {
		return nil, err
	}

	// genesis file
	genFile := config.GenesisFile()
	if util.FileExists(genFile) {
		return nil, errors.New("genesis file already exists")
	}

	// private validator
	privValFile := config.PrivValidatorFile()
	if util.FileExists(privValFile) {
		return nil, errors.New("private validator file already exists")
	}

	privValidator := pv.GenFilePV(privValFile)
	privValidator.Save()

	validator := types.GenesisValidator{
		PubKey: privValidator.GetPubKey(),
		Power:  10,
	}

	genDoc := types.GenesisDoc{
		ChainID:    "default",
		Validators: []types.GenesisValidator{validator},
	}

	err = genDoc.SaveAs(genFile)
	if err != nil {
		return nil, err
	}

	pubKey := [32]byte(validator.PubKey.(crypto.PubKeyEd25519))
	return &loom.Validator{
		PubKey: pubKey[:],
		Power:  validator.Power,
	}, nil
}

// loadFilePV does what tendermint should have done instead of putting exits
// in their code.
func loadFilePV(filePath string) (*pv.FilePV, error) {
	_, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return pv.LoadFilePV(filePath), nil
}

func (b *TendermintBackend) Reset(height uint64) error {
	if height != 0 {
		return errors.New("can only reset back to height 0")
	}
	cfg, err := b.parseConfig()
	if err != nil {
		return err
	}

	err = util.IgnoreErrNotExists(os.RemoveAll(cfg.DBDir()))

	privVal, err := loadFilePV(cfg.PrivValidatorFile())
	if err != nil {
		return err
	}
	resetPrivValidator(privVal, int64(height))
	privVal.Save()

	return nil
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

func (b *TendermintBackend) NodeKey() (string, error) {
	config, err := b.parseConfig()
	if err != nil {
		return "", err
	}

	if nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile()); err != nil {
		return "", err
	} else {
		return string(nodeKey.ID()), nil
	}
}

func (b *TendermintBackend) Destroy() error {
	config, err := b.parseConfig()
	if err != nil {
		return err
	}

	err = util.IgnoreErrNotExists(os.RemoveAll(config.DBDir()))
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
	err = util.IgnoreErrNotExists(os.Remove(config.NodeKeyFile()))
	if err != nil {
		return err
	}

	return nil
}

func (b *TendermintBackend) Start(app abci.Application) error {
	cfg, err := b.parseConfig()
	if err != nil {
		return err
	}
	levelOpt, err := log.TMAllowLevel(b.OverrideCfg.LogLevel)
	if err != nil {
		return err
	}
	logger := log.NewTMFilter(log.Root, levelOpt)
	cfg.BaseConfig.LogLevel = b.OverrideCfg.LogLevel
	privVal, err := loadFilePV(cfg.PrivValidatorFile())
	if err != nil {
		return err
	}

	cfg.P2P.Seeds = b.OverrideCfg.Peers
	cfg.P2P.PersistentPeers = b.OverrideCfg.PersistentPeers

	// Create & start tendermint node
	n, err := node.NewNode(cfg,
		privVal,
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

func (b *TendermintBackend) EventBus() *types.EventBus {
	return b.node.EventBus()
}

func (b *TendermintBackend) RunForever() {
	// Trap signal, run forever.
	b.node.RunForever()
}
