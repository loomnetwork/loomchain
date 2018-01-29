package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	abci "github.com/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/cli"
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

type experimentApp struct {
	abci.BaseApplication
	TxMiddlewares []loom.TxMiddleware
}

func (a *experimentApp) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	var err error
	ctx := context.Background()
	for _, middleware := range a.TxMiddlewares {
		txBytes, err = middleware.Handle(ctx, txBytes)
		if err != nil {
			return abci.ResponseCheckTx{Code: 1, Log: err.Error()}
		}
	}
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

var _ abci.Application = &experimentApp{}

func main() {
	cmd := cli.PrepareMainCmd(StartCmd, "EX", ".")
	cmd.Execute()

	app := &experimentApp{}
	err := startTendermint(app)
	if err != nil {
		panic(err)
	}
}

func startCmd(cmd *cobra.Command, args []string) error {
	app := &experimentApp{
		TxMiddlewares: []loom.TxMiddleware{loom.SignatureTxMiddleware},
	}
	return startTendermint(app)
}

func startTendermint(app abci.Application) error {
	cfg, err := tcmd.ParseConfig()
	if err != nil {
		return err
	}

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
