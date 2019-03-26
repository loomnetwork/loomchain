package deployer_whitelist

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/spf13/cobra"
)

var (
	dwContractName = "deployerwhitelist"
)

func NewDeployCommand() *cobra.Command {
	cmd := cli.ContractCallCommand("deployerwhitelist")
	cmd.Use = "deployer"
	cmd.Short = "Deployer Whitelist CLI"
	cmd.AddCommand(
		addDeployerCmd(),
		getDeployerCmd(),
		listDeployersCmd(),
		removeDeployerCmd(),
	)
	return cmd
}

const addDeployerCmdExample = `
loom deployer add-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857 any
`

func addDeployerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "add-deployer <deployer address> <permission (go|evm|any|none)>",
		Short:   "Add deployer with permision to deployer list",
		Example: addDeployerCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			var perm dwtypes.DeployPermission
			if strings.EqualFold(args[1], "evm") {
				perm = dwtypes.DeployPermission_EVM
			} else if strings.EqualFold(args[1], "go") {
				perm = dwtypes.DeployPermission_GO
			} else if strings.EqualFold(args[1], "any") {
				perm = dwtypes.DeployPermission_ANY
			} else if strings.EqualFold(args[1], "none") {
				perm = dwtypes.DeployPermission_NONE
			} else {
				return fmt.Errorf("Please specify deploy permission (go|evm|any|none)")
			}
			req := &dwtypes.AddDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
				Permission:   perm,
			}

			return cli.CallContract(dwContractName, "AddDeployer", req, nil)
		},
	}
}

const removeDeployerCmdExample = `
loom deployer remove-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func removeDeployerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove-deployer <deployer address>",
		Short:   "Remove deployer from whitelist",
		Example: removeDeployerCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			req := &dwtypes.RemoveDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}

			return cli.CallContract(dwContractName, "RemoveDeployer", req, nil)
		},
	}
}

const getDeployerCmdExample = `
loom deployer get-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func getDeployerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-deployer <deployer address>",
		Short:   "Show current permissions of a deployer",
		Example: getDeployerCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			req := &dwtypes.GetDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}

			var resp dwtypes.GetDeployerResponse
			if err := cli.StaticCallContract(dwContractName, "GetDeployer", req, &resp); err != nil {
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

const listDeployersCmdExample = `
loom deployer list-deployers
`

func listDeployersCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-deployers ",
		Short:   "Display all deployers in whitelist",
		Example: addDeployerCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &dwtypes.ListDeployersRequest{}
			var resp dwtypes.ListDeployersResponse
			if err := cli.StaticCallContract(dwContractName, "ListDeployers", req, &resp); err != nil {
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

// Utils

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
