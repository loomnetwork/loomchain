package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/abci/backend"
	"github.com/loomnetwork/loom/auth"
	"github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/plugin"
	"github.com/loomnetwork/loom/rpc"
	"github.com/loomnetwork/loom/store"
	"github.com/loomnetwork/loom/util"
	"github.com/loomnetwork/loom/vm"

	"github.com/spf13/cobra"
	dbm "github.com/tendermint/tmlibs/db"
)

type Config struct {
	RootDir         string
	DBName          string
	GenesisFile     string
	PluginsDir      string
	QueryServerHost string
}

func (c *Config) fullPath(p string) string {
	full, err := filepath.Abs(path.Join(c.RootDir, p))
	if err != nil {
		panic(err)
	}
	return full
}

func (c *Config) RootPath() string {
	return c.fullPath(c.RootDir)
}

func (c *Config) GenesisPath() string {
	return c.fullPath(c.GenesisFile)
}

func (c *Config) PluginsPath() string {
	return c.fullPath(c.PluginsDir)
}

func DefaultConfig() *Config {
	return &Config{
		RootDir:         ".",
		DBName:          "app",
		GenesisFile:     "genesis.json",
		PluginsDir:      "contracts",
		QueryServerHost: "tcp://127.0.0.1:9999",
	}
}

var RootCmd = &cobra.Command{
	Use:   "loom",
	Short: "Loom blockchain engine",
}

func newInitCommand(backend backend.Backend) *cobra.Command {
	var force bool

	cfg := DefaultConfig()

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the blockchain",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if force {
				err = backend.Destroy()
				if err != nil {
					return err
				}
				destroyDB(cfg.DBName, cfg.RootPath())
			}
			err = backend.Init()
			if err != nil {
				return err
			}

			err = initDB(cfg.DBName, cfg.RootPath())
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force initialization")
	return cmd
}

func newRunCommand(backend backend.Backend) *cobra.Command {
	cfg := DefaultConfig()
	return &cobra.Command{
		Use:   "run [root contract]",
		Short: "Run the blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := plugin.NewManager(cfg.PluginsPath())
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

type genesis struct {
	PluginName string          `json:"plugin"`
	Init       json.RawMessage `json:"init"`
}

func (g *genesis) InitCode() ([]byte, error) {
	body, err := g.Init.MarshalJSON()
	if err != nil {
		return nil, err
	}

	req := &plugin.Request{
		ContentType: plugin.ContentType_JSON,
		Body:        body,
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	pluginCode := &plugin.PluginCode{
		Name:  g.PluginName,
		Input: input,
	}
	return proto.Marshal(pluginCode)
}

func readGenesis(path string) (*genesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(file)

	var gen genesis
	err = dec.Decode(&gen)
	if err != nil {
		return nil, err
	}

	return &gen, nil
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

func loadApp(chainID string, cfg *Config, loader plugin.Loader) (*loom.Application, error) {
	db, err := dbm.NewGoLevelDB(cfg.DBName, cfg.RootPath())
	if err != nil {
		return nil, err
	}

	appStore, err := store.NewIAVLStore(db)
	if err != nil {
		return nil, err
	}

	vmManager := vm.NewManager()
	vmManager.Register(vm.VMType_PLUGIN, func(state loom.State) vm.VM {
		return &plugin.PluginVM{
			Loader: loader,
			State:  state,
		}
	})

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

	initCode, err := gen.InitCode()
	if err != nil {
		return nil, err
	}

	init := func(state loom.State) error {
		vm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
		if err != nil {
			return err
		}

		_, _, err = vm.Create(loom.RootAddress(chainID), initCode)
		return err
	}

	router := loom.NewTxRouter()
	router.Handle(1, deployTxHandler)
	router.Handle(2, callTxHandler)

	return &loom.Application{
		Store: appStore,
		Init:  init,
		TxHandler: loom.MiddlewareTxHandler(
			[]loom.TxMiddleware{
				log.TxMiddleware,
				auth.SignatureTxMiddleware,
				auth.NonceTxMiddleware,
			},
			router,
		),
	}, nil
}

func main() {
	backend := &backend.TendermintBackend{}

	RootCmd.AddCommand(
		newInitCommand(backend),
		newRunCommand(backend),
	)
	err := RootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
