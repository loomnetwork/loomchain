package cli

import (
	"github.com/spf13/cobra"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/abci/backend"
)

func Commands(backend backend.Backend, app *loom.Application) []*cobra.Command {
	return []*cobra.Command{
		NewInitCommand(backend, app),
		NewRunCommand(backend, app),
	}
}

func NewInitCommand(backend backend.Backend, app *loom.Application) *cobra.Command {
	return &cobra.Command{
		Use:   "init [root contract]",
		Short: "Initialize the blockchain",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return backend.Init()
		},
	}
}

func NewRunCommand(backend backend.Backend, app *loom.Application) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backend.Run(app)
		},
	}
}
