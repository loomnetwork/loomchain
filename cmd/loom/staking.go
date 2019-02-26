package main

import (
	"fmt"

	"github.com/loomnetwork/go-loom/builtin/commands"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/spf13/cobra"
)

func newStakingCommand() *cobra.Command {
	cmd := cli.ContractCallCommand("staking")
	cmd.Use = "staking"
	cmd.Short = "Run staking commands"
	cmd.AddCommand(
		StakingListAllDelegationsCmd(),
	)
	return cmd
}

func StakingListAllDelegationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-all-delegations",
		Short: "display the all delegations",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListAllDelegationsResponse
			err := cli.StaticCallContract(commands.DPOSV2ContractName, "ListAllDelegations", &dposv2.ListAllDelegationsRequest{}, &resp)
			if err != nil {
				return err
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}
