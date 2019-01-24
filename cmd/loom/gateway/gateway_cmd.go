// +build evm

package gateway

import (
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	cmd := newRootCommand()
	cmd.AddCommand(
		newMapContractsCommand(),
		newMapAccountsCommand(),
		newQueryAccountCommand(),
		newReplaceOwnerCommand(),
		newGetStateCommand(),
		newAddOracleCommand(),
		newRemoveOracleCommand(),
		newGetOraclesCommand(),
	)
	return cmd
}
