package backend

import (
	abci "github.com/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loom/log"
)

type Backend interface {
	Run(app abci.Application, logger log.Logger) error
}

type TendermintBackend struct {
}

func (b *TendermintBackend) Run(app abci.Application, logger log.Logger) error {
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
