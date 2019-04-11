// +build evm

package gateway

import (
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	cmd := newRootCommand()
	cmd.AddCommand(
		newWithdrawFundsToMainnetCommand(),
		newMapContractsCommand(),
		newMapAccountsCommand(),
		newQueryAccountCommand(),
		newQueryUnclaimedTokensCommand(),
		newReplaceOwnerCommand(),
		newGetStateCommand(),
		newAddOracleCommand(),
		newRemoveOracleCommand(),
		newGetOraclesCommand(),
		newUpdateTrustedValidatorsCommand(),
	)
	return cmd
}
