package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
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

type genesis struct {
	ChainID    string          `json:"chain_id"`
	PluginName string          `json:"plugin"`
	Init       json.RawMessage `json:"init"`
}

func (g *genesis) InitCode() ([]byte, error) {
	body, err := g.Init.MarshalJSON()
	if err != nil {
		return nil, err
	}

	req := &plugin.Request{
		ContentType: plugin.ContentType_PROTOBUF3,
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

func readGenesis() (*genesis, error) {
	file, err := os.Open("genesis.json")
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

	gen, err := readGenesis()
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

		_, _, err = vm.Create(loom.RootAddress(gen.ChainID), initCode)
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
