package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/abci/backend"
	"github.com/loomnetwork/loom/plugins"
	"github.com/loomnetwork/loom/store"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/tendermint/tmlibs/cli"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/abci/backend"
	"github.com/loomnetwork/loom/auth"
	"github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/store"
)

// RootCmd is the entry point for this binary
var RootCmd = &cobra.Command{
	Use:   "ex",
	Short: "A cryptocurrency framework in Golang based on Tendermint-Core",
}

// StartCmd - command to start running the abci app (and tendermint)!
var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start this full node",
	RunE:  startCmd,
}

const dummyTxID = 1

type helloworldHandler struct {
}

func (a *helloworldHandler) ProcessTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	logger := log.Log(state.Context())
	r := loom.TxHandlerResult{}
	tx := &loom.DummyTx{}
	if err := proto.Unmarshal(txBytes, tx); err != nil {
		return r, err
	}
	logger.Debug(fmt.Sprintf("Got DummyTx { key: '%s', value: '%s' }", tx.Key, tx.Val))
	// Run the tx & update the app state as needed.
	saveDummyValue(state, tx.Key, tx.Val)
	saveLastDummyKey(state, tx.Key)
	return r, nil
}

const rootDir = "."

func main() {
	cmd := cli.PrepareMainCmd(StartCmd, "EX", rootDir)
	cmd.Execute()
}

func startCmd(cmd *cobra.Command, args []string) error {
	db, err := dbm.NewGoLevelDB("helloworld", rootDir)
	if err != nil {
		return err
	}

	appStore, err := store.NewIAVLStore(db)
	if err != nil {
		return err
	}

	pluginDir := "out/*.so"

	router := loom.NewTxRouter()
<<<<<<< HEAD:examples/helloworld/helloworld.go
	router.Handle(dummyTxID, &helloworldHandler{})
=======
	plugins.AttachLocalPlugins(pluginDir, router)
	router.Handle(dummyTxID, &experimentHandler{})
>>>>>>> add inprocess plugins:examples/experiment/experiment.go

	//Iterate the plugins and apply routes

	app := &loom.Application{
		Store: appStore,
		TxHandler: loom.MiddlewareTxHandler(
			[]loom.TxMiddleware{
				log.TxMiddleware,
				auth.SignatureTxMiddleware,
				auth.NonceTxMiddleware,
			},
			router,
		),
		QueryHandler: &queryHandler{},
	}

	nodeBackend := &backend.TendermintBackend{}
	return nodeBackend.Run(app)
}

type queryHandler struct {
}

func (q *queryHandler) Handle(state loom.ReadOnlyState, path string, data []byte) ([]byte, error) {
	logger := log.Log(context.TODO())
	logger.Info(fmt.Sprintf("Query received, path: '%s', data: '%v'", path, data))
	var val string
	var err error

	switch path {
	case "app/last-key":
		val, err = loadLastDummyKey(state)
		break
	case "app/dummy":
		val, err = loadDummyValue(state, string(data))
		break
	default:
		return nil, fmt.Errorf("Invalid query for path '%s'", path)
	}

	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func saveLastDummyKey(state loom.State, key string) {
	state.Set([]byte("app/last-key"), []byte(key))
}

func loadLastDummyKey(state loom.ReadOnlyState) (string, error) {
	val := state.Get([]byte("app/last-key"))
	if len(val) == 0 {
		return "", errors.New("last key not set")
	}
	return string(val), nil
}

func saveDummyValue(state loom.State, key, val string) {
	state.Set([]byte("app/dummy/"+key), []byte(val))
}

func loadDummyValue(state loom.ReadOnlyState, key string) (string, error) {
	val := state.Get([]byte("app/dummy/" + key))
	if len(val) == 0 {
		return "", fmt.Errorf("no value stored for key '%s'", key)
	}
	return string(val), nil
}
