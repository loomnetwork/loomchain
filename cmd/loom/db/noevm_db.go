// +build !evm

package db

import (
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
