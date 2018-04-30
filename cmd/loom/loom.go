package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	dbm "github.com/tendermint/tmlibs/db"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/abci/backend"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/rpc"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
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
				"version":     loomchain.FullVersion(),
				"git sha":     loomchain.GitSHA,
				"plugin path": cfg.PluginsPath(),
			})
			return nil
		},
	}
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
			}
			err = backend.Init()
			if err != nil {
				return err
			}

			err = initApp(cfg)
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

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run [root contract]",
		Short: "Run the blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
			if err != nil {
				return err
			}
			backend := initBackend(cfg)
			loader := plugin.NewMultiLoader(
				plugin.NewManager(cfg.PluginsPath()),
				plugin.NewExternalLoader(cfg.PluginsPath()),
			)

			chainID, err := backend.ChainID()
			if err != nil {
				return err
			}
			app, err := loadApp(chainID, cfg, loader)
			if err != nil {
				return err
			}
			if err := backend.Start(app); err != nil {
				return err
			}
			qs := &rpc.QueryServer{
				StateProvider: app,
				ChainID:       chainID,
				Host:          cfg.QueryServerHost,
				Logger:        log.Root.With("module", "query-server"),
				Loader:        loader,
			}
			if err := qs.Start(); err != nil {
				return err
			}
			backend.RunForever()
			return nil
		},
	}
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

func initApp(cfg *Config) error {
	var gen genesis

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

func loadApp(chainID string, cfg *Config, loader plugin.Loader) (*loomchain.Application, error) {
	logger := log.Root
	db, err := dbm.NewGoLevelDB(cfg.DBName, cfg.RootPath())
	if err != nil {
		return nil, err
	}

	appStore, err := store.NewIAVLStore(db)
	if err != nil {
		return nil, err
	}

	var eventDispatcher loomchain.EventDispatcher
	if cfg.EventDispatcherURI != "" {
		logger.Info(fmt.Sprintf("Using event dispatcher for %s\n", cfg.EventDispatcherURI))
		eventDispatcher, err = loomchain.NewEventDispatcher(cfg.EventDispatcherURI)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Using simple log event dispatcher")
		eventDispatcher = events.NewLogEventDispatcher()
	}
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)

	vmManager := vm.NewManager()
	vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) vm.VM {
		return &plugin.PluginVM{
			Loader:       loader,
			State:        state,
			EventHandler: eventHandler,
		}
	})

	if vm.LoomEvmFactory != nil {
		vmManager.Register(vm.VMType_EVM, vm.LoomVmFactory)
	}

	deployTxHandler := &vm.DeployTxHandler{
		Manager: vmManager,
	}

	callTxHandler := &vm.CallTxHandler{
		Manager: vmManager,
	}

	gen, err := readGenesis(cfg.GenesisPath())
	if err != nil {
		return nil, err
	}

	init := func(state loomchain.State) error {
		for _, contractCfg := range gen.Contracts {
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

			_, addr, err := vm.Create(loom.RootAddress(chainID), initCode)
			if err != nil {
				return err
			}

			if contractCfg.Name != "" {
				err = registry.Register(state, contractCfg.Name, addr, addr)
				if err != nil {
					return err
				}
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

	return &loomchain.Application{
		Store: appStore,
		Init:  init,
		TxHandler: loomchain.MiddlewareTxHandler(
			[]loomchain.TxMiddleware{
				loomchain.LogTxMiddleware,
				loomchain.RecoveryTxMiddleware,
				auth.SignatureTxMiddleware,
				auth.NonceTxMiddleware,
			},
			router,
			[]loomchain.PostCommitMiddleware{
				loomchain.LogPostCommitMiddleware,
			},
		),
		EventHandler: eventHandler,
	}, nil
}

func initBackend(cfg *Config) backend.Backend {
	return &backend.TendermintBackend{
		RootPath: path.Join(cfg.RootPath(), "chaindata"),
	}
}

func main() {
	RootCmd.AddCommand(
		newVersionCommand(),
		newEnvCommand(),
		newInitCommand(),
		newResetCommand(),
		newRunCommand(),
		newSpinCommand(),
		newDeployCommand(),
		newCallCommand(),
	)
	err := RootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
