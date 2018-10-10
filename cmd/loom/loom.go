package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/pkg/errors"

	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/spf13/cobra"
	dbm "github.com/tendermint/tendermint/libs/db"
	"golang.org/x/crypto/ed25519"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/abci/backend"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash"
	plasmaOracle "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/oracle"
	gatewaycmd "github.com/loomnetwork/loomchain/cmd/loom/gateway"
	"github.com/loomnetwork/loomchain/eth/polls"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/evm"
	tgateway "github.com/loomnetwork/loomchain/gateway"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	receiptsfactory "github.com/loomnetwork/loomchain/receipts/factory"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/rpc"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/loomnetwork/loomchain/vm"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/rpc/lib/server"

	plasmaConfig "github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/config"
)

var RootCmd = &cobra.Command{
	Use:   "loom",
	Short: "Loom DAppChain",
}

var codeLoaders map[string]ContractCodeLoader

func init() {
	codeLoaders = map[string]ContractCodeLoader{
		"plugin":   &PluginCodeLoader{},
		"truffle":  &TruffleCodeLoader{},
		"solidity": &SolidityCodeLoader{},
		"hex":      &HexCodeLoader{},
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the Loom chain version",
		RunE: func(cmd *cobra.Command, args []string) error {
			println(loomchain.FullVersion())
			return nil
		},
	}
}

func printEnv(env map[string]string) {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		val := env[key]
		fmt.Printf("%s = %s\n", key, val)
	}
}

func newEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Show loom config settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
			if err != nil {
				return err
			}

			printEnv(map[string]string{
				"version":           loomchain.FullVersion(),
				"git sha":           loomchain.GitSHA,
				"plugin path":       cfg.PluginsPath(),
				"query server host": cfg.QueryServerHost,
				"peers":             cfg.Peers,
			})
			return nil
		},
	}
}

type genKeyFlags struct {
	PublicFile string `json:"publicfile"`
	PrivFile   string `json:"privfile"`
}

func newGenKeyCommand() *cobra.Command {
	var flags genKeyFlags
	keygenCmd := &cobra.Command{
		Use:   "genkey",
		Short: "generate a public and private key pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			pub, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				return fmt.Errorf("Error generating key pair: %v", err)
			}
			encoder := base64.StdEncoding
			pubKeyB64 := encoder.EncodeToString(pub[:])
			privKeyB64 := encoder.EncodeToString(priv[:])

			if err := ioutil.WriteFile(flags.PublicFile, []byte(pubKeyB64), 0664); err != nil {
				return fmt.Errorf("Unable to write public key: %v", err)
			}
			if err := ioutil.WriteFile(flags.PrivFile, []byte(privKeyB64), 0664); err != nil {
				return fmt.Errorf("Unable to write private key: %v", err)
			}
			addr := loom.LocalAddressFromPublicKey(pub[:])
			fmt.Printf("local address: %s\n", addr.String())
			fmt.Printf("local address base64: %s\n", encoder.EncodeToString(addr))
			return nil
		},
	}
	keygenCmd.Flags().StringVarP(&flags.PublicFile, "public_key", "a", "", "public key file")
	keygenCmd.Flags().StringVarP(&flags.PrivFile, "private_key", "k", "", "private key file")
	return keygenCmd
}

func newInitCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configs and data",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
			if err != nil {
				return err
			}
			backend := initBackend(cfg)
			if force {
				err = backend.Destroy()
				if err != nil {
					return err
				}
				err = destroyApp(cfg)
				if err != nil {
					return err
				}
				err = destroyReceiptsDB(cfg)
				if err != nil {
					return errors.Wrap(err, "destroy receipt db")
				}
			}
			validator, err := backend.Init()
			if err != nil {
				return err
			}

			err = initApp(validator, cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force initialization")
	return cmd
}

func newResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset the app and blockchain state only",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
			if err != nil {
				return err
			}

			backend := initBackend(cfg)
			err = backend.Reset(0)
			if err != nil {
				return err
			}

			err = resetApp(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

// Generate Or Prints node's ID to the standard output.
func newNodeKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "nodekey",
		Short: "Show node key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
			if err != nil {
				return err
			}
			backend := initBackend(cfg)
			key, err := backend.NodeKey()
			if err != nil {
				fmt.Printf("Error in determining Node Key")
			} else {
				fmt.Printf("%s\n", key)
			}
			return nil
		},
	}
}

func defaultContractsLoader(cfg *Config) plugin.Loader {
	contracts := []goloomplugin.Contract{
		coin.Contract,
		dpos.Contract,
	}
	if cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts, plasma_cash.Contract)
	}
	if cfg.KarmaEnabled {
		contracts = append(contracts, karma.Contract)
	}
	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts, address_mapper.Contract, gateway.Contract, ethcoin.Contract)
	}
	return plugin.NewStaticLoader(contracts...)
}

func newRunCommand() *cobra.Command {
	cfg, err := parseConfig()

	cmd := &cobra.Command{
		Use:   "run [root contract]",
		Short: "Run the blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err != nil {
				return err
			}
			log.Setup(cfg.LoomLogLevel, cfg.LogDestination)
			backend := initBackend(cfg)
			loader := plugin.NewMultiLoader(
				plugin.NewManager(cfg.PluginsPath()),
				plugin.NewExternalLoader(cfg.PluginsPath()),
				defaultContractsLoader(cfg),
			)

			termChan := make(chan os.Signal)
			go func(c <-chan os.Signal, l plugin.Loader) {
				<-c
				l.UnloadContracts()
				os.Exit(0)
			}(termChan, loader)

			signal.Notify(termChan, syscall.SIGHUP,
				syscall.SIGINT,
				syscall.SIGTERM,
				syscall.SIGQUIT)

			chainID, err := backend.ChainID()
			if err != nil {
				return err
			}
			app, receipts, err := loadApp(chainID, cfg, loader, backend)
			if err != nil {
				return err
			}
			if err := backend.Start(app); err != nil {
				return err
			}
			if err := initQueryService(app, chainID, cfg, loader, receipts); err != nil {
				return err
			}

			if err := startGatewayOracle(chainID, cfg.TransferGateway); err != nil {
				return err
			}

			if err := startPlasmaOracle(chainID, cfg.PlasmaCash); err != nil {
				return err
			}

			backend.RunForever()
			return nil
		},
	}
	cmd.Flags().StringVarP(&cfg.Peers, "peers", "p", "", "peers")
	cmd.Flags().StringVar(&cfg.PersistentPeers, "persistent-peers", "", "persistent peers")
	return cmd
}

func recovery() {
	if r := recover(); r != nil {
		log.Error("caught RPC proxy exception, exiting", r)
		os.Exit(1)
	}
}

func startPlasmaOracle(chainID string, cfg *plasmaConfig.PlasmaCashSerializableConfig) error {
	plasmaCfg, err := plasmaConfig.LoadSerializableConfig(chainID, cfg)
	if err != nil {
		return err
	}

	if !plasmaCfg.OracleEnabled {
		return nil
	}

	oracle := plasmaOracle.NewOracle(plasmaCfg.OracleConfig)
	err = oracle.Init()
	if err != nil {
		return err
	}

	oracle.Run()

	return nil
}

func startGatewayOracle(chainID string, cfg *tgateway.TransferGatewayConfig) error {
	if !cfg.OracleEnabled {
		return nil
	}

	orc, err := tgateway.CreateOracle(cfg, chainID)
	if err != nil {
		return err
	}
	go orc.RunWithRecovery()
	return nil
}

func initDB(name, dir string) error {
	dbPath := filepath.Join(dir, name+".db")
	if util.FileExists(dbPath) {
		return errors.New("db already exists")
	}

	return nil
}

func destroyDB(name, dir string) error {
	dbPath := filepath.Join(dir, name+".db")
	return os.RemoveAll(dbPath)
}

func resetApp(cfg *Config) error {
	return destroyDB(cfg.DBName, cfg.RootPath())
}

func initApp(validator *loom.Validator, cfg *Config) error {
	gen, err := defaultGenesis(cfg, validator)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(cfg.GenesisPath(), os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	err = enc.Encode(gen)
	if err != nil {
		return err
	}

	err = initDB(cfg.DBName, cfg.RootPath())
	if err != nil {
		return err
	}

	return nil
}

func destroyApp(cfg *Config) error {
	err := util.IgnoreErrNotExists(os.Remove(cfg.GenesisPath()))
	if err != nil {
		return err
	}
	return resetApp(cfg)
}

func destroyReceiptsDB(cfg *Config) error {
	receiptVer, err := receiptsfactory.ReceiptHandlerVersionFromInt(cfg.ReceiptsVersion)
	if err != nil {
		return errors.Wrap(err, "find receipt handler vers")
	}
	receiptHandler, err := receiptsfactory.NewReceiptHandlerFactory(receiptVer, &loomchain.DefaultEventHandler{})
	if err != nil {
		return errors.Wrap(err, "new receipt fandler factory")
	}
	return receiptHandler.ClearData()
}

func loadApp(chainID string, cfg *Config, loader plugin.Loader, b backend.Backend) (*loomchain.Application, receipts.ReceiptHandler, error) {
	logger := log.Root
	db, err := dbm.NewGoLevelDB(cfg.DBName, cfg.RootPath())
	if err != nil {
		return nil, nil, err
	}

	var appStore store.VersionedKVStore
	if cfg.LogStateDB {
		appStore, err = store.NewLogStore(db)
	} else {
		appStore, err = store.NewIAVLStore(db)
	}
	if err != nil {
		return nil, nil, err
	}

	var eventDispatcher loomchain.EventDispatcher
	if cfg.EventDispatcherURI != "" {
		logger.Info(fmt.Sprintf("Using event dispatcher for %s\n", cfg.EventDispatcherURI))
		eventDispatcher, err = loomchain.NewEventDispatcher(cfg.EventDispatcherURI)
		if err != nil {
			return nil, nil, err
		}
	} else {
		logger.Info("Using simple log event dispatcher")
		eventDispatcher = events.NewLogEventDispatcher()
	}
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)

	// TODO: It shouldn't be possible to change the registry version via config after the first run,
	//       changing it from that point on should require a special upgrade tx that stores the
	//       new version in the app store.
	regVer, err := registry.RegistryVersionFromInt(cfg.RegistryVersion)
	if err != nil {
		return nil, nil, err
	}
	createRegistry, err := registry.NewRegistryFactory(regVer)
	if err != nil {
		return nil, nil, err
	}

	receiptVer, err := receiptsfactory.ReceiptHandlerVersionFromInt(cfg.ReceiptsVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "find receipt handler vers")
	}
	receiptHandler, err := receiptsfactory.NewReceiptHandlerFactory(receiptVer, eventHandler)
	if err != nil {
		return nil, nil, errors.Wrap(err, "new receipt fandler factory")
	}

	var newABMFactory plugin.NewAccountBalanceManagerFactoryFunc
	if evm.EVMEnabled && cfg.EVMAccountsEnabled {
		newABMFactory = plugin.NewAccountBalanceManagerFactory
	}

	vmManager := vm.NewManager()
	vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return plugin.NewPluginVM(
			loader,
			state,
			createRegistry(state),
			eventHandler,
			log.Default,
			newABMFactory,
			receiptHandler,
		), nil
	})

	if evm.EVMEnabled {
		vmManager.Register(vm.VMType_EVM, func(state loomchain.State) (vm.VM, error) {
			var createABM evm.AccountBalanceManagerFactoryFunc
			var err error

			if newABMFactory != nil {
				pvm := plugin.NewPluginVM(
					loader,
					state,
					createRegistry(state),
					eventHandler,
					log.Default,
					newABMFactory,
					receiptHandler,
				)
				createABM, err = newABMFactory(pvm)
				if err != nil {
					return nil, err
				}
			}
			//TODO pass in evm debug state
			return evm.NewLoomVm(state, eventHandler, receiptHandler, createABM, false), nil
		})
	}
	evm.LogEthDbBatch = cfg.LogEthDbBatch

	deployTxHandler := &vm.DeployTxHandler{
		Manager:        vmManager,
		CreateRegistry: createRegistry,
	}

	callTxHandler := &vm.CallTxHandler{
		Manager: vmManager,
	}

	gen, err := readGenesis(cfg.GenesisPath())
	if err != nil {
		return nil, nil, err
	}

	rootAddr := loom.RootAddress(chainID)
	init := func(state loomchain.State) error {
		registry := createRegistry(state)
		evm.AddLoomPrecompiles()
		for i, contractCfg := range gen.Contracts {
			vmType := contractCfg.VMType()
			vm, err := vmManager.InitVM(vmType, state)
			if err != nil {
				return err
			}

			loader := codeLoaders[contractCfg.Format]
			initCode, err := loader.LoadContractCode(
				contractCfg.Location,
				contractCfg.Init,
			)
			if err != nil {
				return err
			}

			callerAddr := plugin.CreateAddress(rootAddr, uint64(i))
			_, addr, err := vm.Create(callerAddr, initCode, loom.NewBigUIntFromInt(0))
			if err != nil {
				return err
			}

			err = registry.Register(contractCfg.Name, addr, addr)
			if err != nil {
				return err
			}

			logger.Info("Deployed contract",
				"vm", contractCfg.VMTypeName,
				"location", contractCfg.Location,
				"name", contractCfg.Name,
				"address", addr,
			)
		}
		return nil
	}

	router := loomchain.NewTxRouter()
	router.Handle(1, deployTxHandler)
	router.Handle(2, callTxHandler)

	txMiddleWare := []loomchain.TxMiddleware{
		loomchain.LogTxMiddleware,
		loomchain.RecoveryTxMiddleware,
		auth.SignatureTxMiddleware,
	}

	if cfg.KarmaEnabled {
		txMiddleWare = append(txMiddleWare, throttle.GetKarmaMiddleWare(
			cfg.KarmaEnabled,
			cfg.KarmaMaxCallCount,
			cfg.KarmaSessionDuration,
			cfg.KarmaMaxDeployCount,
			registry.RegistryVersion(cfg.RegistryVersion),
		))
	}

	txMiddleWare = append(txMiddleWare, auth.NonceTxMiddleware)

	oracle, err := loom.ParseAddress(cfg.Oracle)
	if err != nil {
		oracle = loom.Address{}
	}
	txMiddleWare = append(txMiddleWare, throttle.GetThrottleTxMiddleWare(
		cfg.DeployEnabled,
		cfg.CallEnabled,
		oracle,
	))

	txMiddleWare = append(txMiddleWare, loomchain.NewInstrumentingTxMiddleware())

	return &loomchain.Application{
		Store: appStore,
		Init:  init,
		TxHandler: loomchain.MiddlewareTxHandler(
			txMiddleWare,
			router,
			[]loomchain.PostCommitMiddleware{
				loomchain.LogPostCommitMiddleware,
			},
		),
		UseCheckTx:   cfg.UseCheckTx,
		EventHandler: eventHandler,
	}, receiptHandler, nil
}

func initBackend(cfg *Config) backend.Backend {
	ovCfg := &backend.OverrideConfig{
		LogLevel:          cfg.BlockchainLogLevel,
		Peers:             cfg.Peers,
		PersistentPeers:   cfg.PersistentPeers,
		ChainID:           cfg.ChainID,
		RPCListenAddress:  cfg.RPCListenAddress,
		RPCProxyPort:      cfg.RPCProxyPort,
		CreateEmptyBlocks: cfg.CreateEmptyBlocks,
	}
	return &backend.TendermintBackend{
		RootPath:    path.Join(cfg.RootPath(), "chaindata"),
		OverrideCfg: ovCfg,
	}
}

func initQueryService(app *loomchain.Application, chainID string, cfg *Config, loader plugin.Loader, receiptHandler receipts.ReceiptHandler) error {
	// metrics
	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "query_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "query_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	regVer, err := registry.RegistryVersionFromInt(cfg.RegistryVersion)
	if err != nil {
		return err
	}
	createRegistry, err := registry.NewRegistryFactory(regVer)
	if err != nil {
		return err
	}

	var newABMFactory plugin.NewAccountBalanceManagerFactoryFunc
	if evm.EVMEnabled && cfg.EVMAccountsEnabled {
		newABMFactory = plugin.NewAccountBalanceManagerFactory
	}

	qs := &rpc.QueryServer{
		StateProvider:    app,
		ChainID:          chainID,
		Loader:           loader,
		Subscriptions:    app.EventHandler.SubscriptionSet(),
		EthSubscriptions: app.EventHandler.EthSubscriptionSet(),
		EthPolls:         *polls.NewEthSubscriptions(),
		CreateRegistry:   createRegistry,
		NewABMFactory:    newABMFactory,
		ReceiptHandler:   receiptHandler,
		RPCListenAddress: cfg.RPCListenAddress,
	}
	bus := &rpc.QueryEventBus{
		Subs:    *app.EventHandler.SubscriptionSet(),
		EthSubs: *app.EventHandler.EthSubscriptionSet(),
	}
	// query service
	var qsvc rpc.QueryService
	{
		qsvc = qs
		qsvc = rpc.NewInstrumentingMiddleWare(requestCount, requestLatency, qsvc)
	}
	logger := log.Root.With("module", "query-server")
	err = rpc.RPCServer(qsvc, logger, bus, cfg.RPCBindAddress)
	if err != nil {
		return err
	}

	// run http server
	//TODO we should remove queryserver once backwards compatibility is no longer needed
	handler := rpc.MakeQueryServiceHandler(qsvc, logger, bus)
	_, err = rpcserver.StartHTTPServer(cfg.QueryServerHost, handler, logger, rpcserver.Config{MaxOpenConnections: 0})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	karmaCmd := newContractCmd(KarmaContractName)
	RootCmd.AddCommand(
		newVersionCommand(),
		newEnvCommand(),
		newInitCommand(),
		newResetCommand(),
		newRunCommand(),
		newSpinCommand(),
		newDeployCommand(),
		newCallCommand(),
		newGenKeyCommand(),
		newNodeKeyCommand(),
		newStaticCallCommand(),
		newGetBlocksByNumber(),
		karmaCmd,
		gatewaycmd.NewGatewayCommand(),
	)
	AddKarmaMethods(karmaCmd)

	err := RootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
