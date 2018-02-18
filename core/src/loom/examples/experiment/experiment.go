package main

import (
	"fmt"
	"os"

	"github.com/loomnetwork/test-sdk/store"
	samplesdk "github.com/loomnetwork/test-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/cli"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	"loom"
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

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
)

type experimentHandler struct {
}

func (a *experimentHandler) Handle(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	r := loom.TxHandlerResult{}
	tx := &loom.DummyTx{}
	if err := proto.Unmarshal(txBytes, tx); err != nil {
		return r, err
	}
	logger.Debug(fmt.Sprintf("Got DummyTx with value '%s'", tx.GetVal()))
	return r, nil
}

const rootDir = "."

func main() {
	cmd := cli.PrepareMainCmd(StartCmd, "EX", rootDir)
	cmd.Execute()
}

func startCmd(cmd *cobra.Command, args []string) error {
	mainStoreKey := testsdk.NewKVStoreKey("main")
	db, err := dbm.NewGoLevelDB("experiment", rootDir)
	if err != nil {
		return err
	}
	store := store.NewCommitMultiStore(db)
	store.MountStoreWithDB(mainStoreKey, testsdk.StoreTypeIAVL, db)
	err = store.LoadLatestVersion()
	if err != nil {
		return err
	}

	app := &loom.Application{
		TxHandler: loom.MiddlewareTxHandler(
			[]loom.TxMiddleware{
				loom.SignatureTxMiddleware,
			},
			&experimentHandler{},
		),
		QueryHandler: &queryLogger{},
		Store:        store.GetCommitKVStore(mainStoreKey),
	}
	return startTendermint(app)
}

func startTendermint(app abci.Application) error {
	cfg, err := tcmd.ParseConfig()
	if err != nil {
		return err
	}

	cfg.Consensus.CreateEmptyBlocks = false

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

	// Trap signal, run forever.
	n.RunForever()
	return nil
}

type queryLogger struct {
}

func (q *queryLogger) Handle(state loom.State, path string, data []byte) ([]byte, error) {
	logger.Info(fmt.Sprintf("Query received, path: '%s', data: '%v'", path, data))
	return nil, fmt.Errorf("Invalid query for path '%s'", path)
}
