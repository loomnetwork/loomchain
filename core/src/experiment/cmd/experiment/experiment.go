package main

import (
	"context"
	"errors"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/ed25519"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/cli"
	"github.com/tendermint/tmlibs/log"

	"experiment"
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

type TxMiddleware interface {
	Handle(ctx context.Context, txBytes []byte) ([]byte, error)
}

type TxMiddlewareFunc func(ctx context.Context, txBytes []byte) ([]byte, error)

func (f TxMiddlewareFunc) Handle(ctx context.Context, txBytes []byte) ([]byte, error) {
	return f(ctx, txBytes)
}

func SignatureTxMiddleware(ctx context.Context, txBytes []byte) ([]byte, error) {
	var tx experiment.SignedTx

	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return nil, err
	}

	for _, signer := range tx.Signers {
		var pubKey [ed25519.PublicKeySize]byte
		var sig [ed25519.SignatureSize]byte

		if len(signer.PublicKey) != len(pubKey) {
			return nil, errors.New("invalid public key length")
		}

		if len(signer.Signature) != len(sig) {
			return nil, errors.New("invalid signature length")
		}

		copy(pubKey[:], signer.PublicKey)
		copy(sig[:], signer.Signature)

		if !ed25519.Verify(&pubKey, tx.Inner, &sig) {
			return nil, errors.New("invalid signature")
		}

		// TODO: set some context
	}

	return tx.Inner, nil
}

type experimentApp struct {
	abci.BaseApplication
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
	app := &experimentApp{}
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
