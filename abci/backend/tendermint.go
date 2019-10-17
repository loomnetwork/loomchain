package backend

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/loomnetwork/loomchain/fnConsensus"

	pv "github.com/loomnetwork/loomchain/privval"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	dbm "github.com/tendermint/tendermint/libs/db"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
	abci_server "github.com/tendermint/tendermint/abci/server"
	tmcmn "github.com/tendermint/tendermint/libs/common"

	tmLog "github.com/tendermint/tendermint/libs/log"
)

func CreateFnConsensusReactor(
	chainID string,
	privVal types.PrivValidator,
	fnRegistry fnConsensus.FnRegistry,
	cfg *cfg.Config,
	logger tmLog.Logger,
	cachedDBProvider node.DBProvider,
	reactorConfig *fnConsensus.ReactorConfigParsable,
) (*fnConsensus.FnConsensusReactor, error) {
	fnConsensusDB, err := cachedDBProvider(&node.DBContext{ID: "fnConsensus", Config: cfg})
	if err != nil {
		return nil, err
	}

	tmStateDB, err := cachedDBProvider(&node.DBContext{ID: "state", Config: cfg})
	if err != nil {
		return nil, err
	}

	fnConsensusReactor, err := fnConsensus.NewFnConsensusReactor(
		chainID, privVal, fnRegistry, fnConsensusDB, tmStateDB, reactorConfig,
	)
	if err != nil {
		return nil, err
	}

	fnConsensusReactor.SetLogger(logger.With("module", "FnConsensus"))
	return fnConsensusReactor, nil
}

func CreateNewCachedDBProvider(config *cfg.Config) (node.DBProvider, error) {
	// Let's not intefere with other db's creation, unless required
	dbsNeedToCache := []string{
		"state",
		"fnConsensus",
	}

	cachedDBMap := make(map[string]dbm.DB)

	for _, dbNeedToCache := range dbsNeedToCache {
		db, err := node.DefaultDBProvider(&node.DBContext{ID: dbNeedToCache, Config: config})
		if err != nil {
			return nil, err
		}

		cachedDBMap[dbNeedToCache] = db
	}

	return func(ctx *node.DBContext) (dbm.DB, error) {
		cachedDB, ok := cachedDBMap[ctx.ID]
		if !ok {
			return node.DefaultDBProvider(ctx)
		}
		return cachedDB, nil
	}, nil
}

type Backend interface {
	ChainID() (string, error)
	Init() (*loom.Validator, error)
	Reset(height uint64) error
	Destroy() error
	Start(app abci.Application) error
	RunForever()
	GenesisValidators() []*loom.Validator
	// IsValidator checks if this node is currently a validator.
	IsValidator() bool
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
	SocketPath        string
	socketServer      tmcmn.Service
	genesisValidators []*loom.Validator

	FnRegistry fnConsensus.FnRegistry
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
	LogLevel                 string
	Peers                    string
	PersistentPeers          string
	ChainID                  string
	RPCListenAddress         string
	RPCProxyPort             int32
	P2PPort                  int32
	CreateEmptyBlocks        bool
	HsmConfig                *hsmpv.HsmConfig
	FnConsensusReactorConfig *fnConsensus.ReactorConfigParsable
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

	pubKey := [ed25519.PubKeyEd25519Size]byte(validator.PubKey.(ed25519.PubKeyEd25519))
	return &loom.Validator{
		PubKey: pubKey[:],
		Power:  validator.Power,
	}, nil
}

// Return validators list from genesis file
func (b *TendermintBackend) GenesisValidators() []*loom.Validator {
	return b.genesisValidators
}

// IsValidator checks if the node is currently a validator.
func (b *TendermintBackend) IsValidator() bool {
	privVal := b.node.PrivValidator()
	if privVal == nil {
		return false
	}

	// consensus state may be unavailable while the node is catching up
	cs := b.node.ConsensusState()
	if cs == nil {
		return false
	}

	privValAddr := privVal.GetPubKey().Address()
	_, validators := cs.GetValidators()
	for _, validator := range validators {
		if bytes.Equal(privValAddr, validator.Address) {
			return true
		}
	}
	return false
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
	if err != nil {
		return err
	}
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

	return pv.NewEd25519Signer(privVal), nil
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

	//Load genesis validators
	genDoc, err := types.GenesisDocFromFile(cfg.GenesisFile())
	if err != nil {
		return err
	}
	validators := make([]*loom.Validator, 0)
	for _, validator := range genDoc.Validators {
		pubKey := [ed25519.PubKeyEd25519Size]byte(validator.PubKey.(ed25519.PubKeyEd25519))
		validators = append(validators, &loom.Validator{
			PubKey: pubKey[:],
			Power:  validator.Power,
		})
	}
	b.genesisValidators = validators

	if !cmn.FileExists(cfg.NodeKeyFile()) {
		return errors.New("failed to locate local node p2p key file")
	}

	nodeKey, err := p2p.LoadNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}

	cfg.P2P.Seeds = b.OverrideCfg.Peers
	cfg.P2P.PersistentPeers = b.OverrideCfg.PersistentPeers

	nodeLogger := logger.With("module", "node")
	reactorRegistrationRequests := make([]*node.ReactorRegistrationRequest, 0)

	dbProvider := node.DefaultDBProvider

	if b.FnRegistry != nil {
		dbProvider, err = CreateNewCachedDBProvider(cfg)
		if err != nil {
			return err
		}

		reactorConfig := b.OverrideCfg.FnConsensusReactorConfig
		if reactorConfig.IsValidator {
			fnConsensusReactor, err := CreateFnConsensusReactor(b.OverrideCfg.ChainID, privVal, b.FnRegistry, cfg, nodeLogger,
				dbProvider, b.OverrideCfg.FnConsensusReactorConfig)
			if err != nil {
				return err
			}

			reactorRegistrationRequests = append(reactorRegistrationRequests, &node.ReactorRegistrationRequest{
				Name:    "FNCONSENSUS",
				Reactor: fnConsensusReactor,
			})
		}
	}

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
			dbProvider,
			node.DefaultMetricsProvider(cfg.Instrumentation),
			logger.With("module", "node"),
			reactorRegistrationRequests,
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
