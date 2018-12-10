package db

import (
	"github.com/loomnetwork/loomchain/cmd/loom/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/spf13/cobra"
)

func newPruneDBCommand() *cobra.Command {
	var numVersions int64
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Deletes older tree versions from app.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}
			return store.PruneDatabase(cfg.DBName, cfg.RootPath(), numVersions)
		},
	}
	flags := cmd.Flags()
	flags.Int64VarP(&numVersions, "versions", "n", 0, "Number of tree versions to prune")
	return cmd
}

func newCompactDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compacts app.db to reclaim disk space",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}
			return store.CompactDatabase(cfg.DBName, cfg.RootPath())
		},
	}
	return cmd
}
