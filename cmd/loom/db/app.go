package db

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/store"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func newPruneDBCommand() *cobra.Command {
	var numVersions int64
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Deletes older tree versions from app.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.ParseConfig()
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
			cfg, err := config.ParseConfig()
			if err != nil {
				return err
			}
			return store.CompactDatabase(cfg.DBName, cfg.RootPath())
		},
	}
	return cmd
}

func newGetAppHeightCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app-height <path/to/app.db>",
		Short: "Show the last height of app.db",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)

			db, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return err
			}
			defer db.Close()

			iavlStore, err := store.NewIAVLStore(db, 0, 0, 0)
			if err != nil {
				return err
			}

			fmt.Printf("app.db at height: %d \n", iavlStore.Version())
			return nil
		},
	}
	return cmd
}
