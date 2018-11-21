package main

import (
	"encoding/base64"
	"io/ioutil"
	"strings"

	"github.com/loomnetwork/loomchain/privval/auth"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database Maintenance",
	}
	cmd.AddCommand(
		newPruneDBCommand(),
		newCompactDBCommand(),
	)
	return cmd
}

func newPruneDBCommand() *cobra.Command {
	var numVersions int64
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Deletes older tree versions from app.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig()
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
			cfg, err := parseConfig()
			if err != nil {
				return err
			}
			return store.CompactDatabase(cfg.DBName, cfg.RootPath())
		},
	}
	return cmd
}

// Loads the given DAppChain private key.
// privateKeyPath can either be a base64-encoded string representing the key, or the path to a file
// containing the base64-encoded key, in the latter case the path must be prefixed by file://
// (e.g. file://path/to/some.key)
func getDAppChainSigner(privateKeyPath string) (auth.Signer, error) {
	keyStr := privateKeyPath
	if strings.HasPrefix(privateKeyPath, "file://") {
		b64, err := ioutil.ReadFile(strings.TrimPrefix(privateKeyPath, "file://"))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load key file")
		}
		keyStr = string(b64)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 key file")
	}

	return auth.NewSigner(keyBytes), nil
}
