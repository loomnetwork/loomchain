package main

import (
	"fmt"
	"os"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/abci/backend"
	"github.com/loomnetwork/loom/auth"
	"github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/plugin"
	"github.com/loomnetwork/loom/store"
	"github.com/loomnetwork/loom/vm"

	"github.com/spf13/cobra"
	dbm "github.com/tendermint/tmlibs/db"
)

const rootDir = "."

var RootCmd = &cobra.Command{
	Use:   "loom",
	Short: "Loom blockchain engine",
}

func newInitCommand(backend backend.Backend) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the blockchain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backend.Init()
		},
	}
}

func newRunCommand(backend backend.Backend) *cobra.Command {
	return &cobra.Command{
		Use:   "run [root contract]",
		Short: "Run the blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			return backend.Run(app)
		},
	}
}

func loadApp() (*loom.Application, error) {
	db, err := dbm.NewGoLevelDB("app", rootDir)
	if err != nil {
		return nil, err
	}

	appStore, err := store.NewIAVLStore(db)
	if err != nil {
		return nil, err
	}

	loader := plugin.NewManager("./contracts")

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

	router := loom.NewTxRouter()
	router.Handle(1, deployTxHandler)
	router.Handle(2, callTxHandler)

	return &loom.Application{
		Store: appStore,
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
