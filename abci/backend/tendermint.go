package backend

import (
	"errors"
	"fmt"
	"os"
	"time"

	pv "github.com/loomnetwork/loomchain/privval"
	"github.com/loomnetwork/loomchain/privval/auth"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
	abci_server "github.com/tendermint/tendermint/abci/server"
	tmcmn "github.com/tendermint/tendermint/libs/common"
)

type Backend interface {
	ChainID() (string, error)
	Init() (*loom.Validator, error)
	Reset(height uint64) error
	Destroy() error
	Start(app abci.Application) error
	RunForever()
	NodeKey() (string, error)
	// Returns the tx signer used by this node to sign txs it creates
	NodeSigner() (auth.Signer, error)
	// Returns the TCP or UNIX socket address the backend RPC server listens on
	RPCAddress() (string, error)
	EventBus() *types.EventBus // TODO: doesn't seem to be used, remove it
}

type TendermintBackend struct {
	RootPath    string
	node        *node.Node
	OverrideCfg *OverrideConfig
	// Unix socket path to serve ABCI app at
	SocketPath   string
	socketServer tmcmn.Service
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
	//Add overrides here
	if b.OverrideCfg.RPCListenAddress != "" {
		conf.RPC.ListenAddress = b.OverrideCfg.RPCListenAddress
	}
	conf.ProxyApp = fmt.Sprintf("tcp://127.0.0.1:%d", b.OverrideCfg.RPCProxyPort)
	conf.Consensus.CreateEmptyBlocks = b.OverrideCfg.CreateEmptyBlocks
	conf.Mempool.WalPath = "data/mempool.wal"

	cfg.EnsureRoot(b.RootPath)
	return conf, err
}

type OverrideConfig struct {
	LogLevel          string
	Peers             string
	PersistentPeers   string
	ChainID           string
	RPCListenAddress  string
	RPCProxyPort      int32
	P2PPort           int32
	CreateEmptyBlocks bool
	HsmConfig         *hsmpv.HsmConfig
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

	privValidator, err := pv.GenPrivVal(privValFile, b.OverrideCfg.HsmConfig)
	if err != nil {
		return nil, err
	}
	privValidator.Save()

	validator := types.GenesisValidator{
		PubKey: privValidator.GetPubKey(),
		Power:  10,
	}

	chainID := "default"
	if b.OverrideCfg.ChainID != "" {
		chainID = b.OverrideCfg.ChainID
	}
	genDoc := types.GenesisDoc{
		ChainID:     chainID,
		Validators:  []types.GenesisValidator{validator},
		GenesisTime: time.Now(), //Note this has to match on the entire cluster, TODO probably should move this to loom.yaml
	}

	err = genDoc.ValidateAndComplete()
	if err != nil {
		return nil, err
	}

	err = genDoc.SaveAs(genFile)
	if err != nil {
		return nil, err
	}

	_, err = b.NodeKey()
	if err != nil {
		return nil, err
	}

	return &loom.Validator{
		PubKey: privValidator.GetPubKeyBytes(validator.PubKey),
		Power:  validator.Power,
	}, nil
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

	privVal, err := pv.LoadPrivVal(cfg.PrivValidatorFile(), b.OverrideCfg.HsmConfig)
	if err != nil {
		return err
	}
	privVal.Reset(int64(height))
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

func (b *TendermintBackend) NodeSigner() (auth.Signer, error) {
	cfg, err := b.parseConfig()
	if err != nil {
		return nil, err
	}

	privVal, err := pv.LoadPrivVal(cfg.PrivValidatorFile(), b.OverrideCfg.HsmConfig)
	if err != nil {
		return nil, err
	}

	return pv.NewPrivValSigner(privVal), nil
}

func (b *TendermintBackend) RPCAddress() (string, error) {
	cfg, err := b.parseConfig()
	if err != nil {
		return "", err
	}

	return cfg.RPC.ListenAddress, nil
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
	privVal, err := pv.LoadPrivVal(cfg.PrivValidatorFile(), b.OverrideCfg.HsmConfig)
	if err != nil {
		return err
	}

	if !cmn.FileExists(cfg.NodeKeyFile()) {
		return errors.New("failed to locate local node p2p key file")
	}

	nodeKey, err := p2p.LoadNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}

	cfg.P2P.Seeds = b.OverrideCfg.Peers
	cfg.P2P.PersistentPeers = b.OverrideCfg.PersistentPeers

	if b.SocketPath != "" {
		s := abci_server.NewSocketServer(b.SocketPath, app)
		s.SetLogger(logger.With("module", "abci-server"))
		if err := s.Start(); err != nil {
			return err
		}
		b.socketServer = s
	} else {
		// Create & start tendermint node
		n, err := node.NewNode(
			cfg,
			privVal,
			nodeKey,
			proxy.NewLocalClientCreator(app),
			node.DefaultGenesisDocProviderFunc(cfg),
			node.DefaultDBProvider,
			node.DefaultMetricsProvider(cfg.Instrumentation),
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
	}
	return nil
}

func (b *TendermintBackend) EventBus() *types.EventBus {
	return b.node.EventBus()
}

func (b *TendermintBackend) RunForever() {
	cmn.TrapSignal(func() {
		if (b.node != nil) && b.node.IsRunning() {
			b.node.Stop()
		}
		if (b.socketServer != nil) && b.socketServer.IsRunning() {
			b.socketServer.Stop()
		}
	})
}
