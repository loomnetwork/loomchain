package userdeployerwhitelist

import (
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/spf13/cobra"
)

var (
	dwContractName = "user-deployer-whitelist"
)

type deployerInfo struct {
	Address string
	Flags   string
}

func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userdeployer <command>",
		Short: "User Deployer Whitelist CLI",
	}

	cmd.AddCommand(
		addUserDeployerCmd(),
		getUserDeployersCmd(),
		getDeployedContractsCmd(),
	)
	return cmd
}

const addUserDeployerCmdExample = `
loom userdeployer add 0x7262d4c97c7B93937E4810D289b7320e9dA82857 
`

func addUserDeployerCmd() *cobra.Command {
	var flag cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "add <deployer address>",
		Short:   "Add deployer corresponding to the user with EVM permision to deployer list",
		Example: addUserDeployerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAddress(args[0])
			if err != nil {
				return err
			}
			cmd.SilenceUsage = true
			req := &udwtypes.WhitelistUserDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}
			return cli.CallContractWithFlags(&flag, dwContractName, "AddUserDeployer", req, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flag)
	return cmd
}

const getUserDeployersCmdExample = `
loom userdeployer getdeployers
`

func getUserDeployersCmd() *cobra.Command {
	var flag cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "getdeployers",
		Short:   "Get deployer objects corresponding to the user with EVM permision to deployer list",
		Example: getUserDeployersCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// addr, err := parseAddress(args[0])
			// if err != nil {
			// 	return err
			// }
			cmd.SilenceUsage = true
			req := &udwtypes.GetDeployedContractsRequest{}
			return cli.CallContractWithFlags(&flag, dwContractName, "GetUserDeployers", req, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flag)
	return cmd
}

const getDeployedContractsCmdExample = `
loom userdeployer getContracts 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func getDeployedContractsCmd() *cobra.Command {
	var flag cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "getContracts <deployer address>",
		Short:   "Contract addresses deployed by particular deployer",
		Example: getDeployedContractsCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAddress(args[0])
			if err != nil {
				return err
			}
			cmd.SilenceUsage = true
			req := &udwtypes.GetDeployedContractsRequest{
				DeployerAddr: addr.MarshalPB(),
			}
			return cli.CallContractWithFlags(&flag, dwContractName, "GetDeployedContracts", req, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flag)
	return cmd
}
