package deployer_whitelist

import (
	"encoding/json"
	"fmt"
	"strings"

	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/spf13/cobra"
)

var (
	dwContractName = "deployerwhitelist"
)

type deployerInfo struct {
	Address string
	Flags   string
}

func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployer <command>",
		Short: "Deployer Whitelist CLI",
	}

	cmd.AddCommand(
		addDeployerCmd(),
		getDeployerCmd(),
		listDeployersCmd(),
		removeDeployerCmd(),
	)
	return cmd
}

const addDeployerCmdExample = `
loom deployer add 0x7262d4c97c7B93937E4810D289b7320e9dA82857 all
`

func addDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "add <deployer address> <permission (go|evm|migration|all)>",
		Short:   "Add a deployer with permission to the deployer list",
		Example: addDeployerCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			var permFlags uint32
			if strings.EqualFold(args[1], "evm") {
				permFlags = uint32(dw.AllowEVMDeployFlag)
			} else if strings.EqualFold(args[1], "go") {
				permFlags = uint32(dw.AllowGoDeployFlag)
			} else if strings.EqualFold(args[1], "migration") {
				permFlags = uint32(dw.AllowMigrationFlag)
			} else if strings.EqualFold(args[1], "all") {
				permFlags = dw.PackFlags(
					uint32(dw.AllowEVMDeployFlag), uint32(dw.AllowGoDeployFlag),
					uint32(dw.AllowMigrationFlag),
				)
			} else {
				return fmt.Errorf("Please specify permissions (go|evm|any) for the new deployer")
			}

			cmd.SilenceUsage = true

			req := &dwtypes.AddDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
				Flags:        permFlags,
			}

			return cli.CallContractWithFlags(&flags, dwContractName, "AddDeployer", req, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const removeDeployerCmdExample = `
loom deployer remove 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func removeDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "remove <deployer address>",
		Short:   "Remove deployer from whitelist",
		Example: removeDeployerCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			req := &dwtypes.RemoveDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}

			return cli.CallContractWithFlags(&flags, dwContractName, "RemoveDeployer", req, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getDeployerCmdExample = `
loom deployer get 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func getDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get <deployer address>",
		Short:   "Show current permissions for a deployer",
		Example: getDeployerCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			req := &dwtypes.GetDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}

			var resp dwtypes.GetDeployerResponse
			if err := cli.StaticCallContractWithFlags(&flags, dwContractName, "GetDeployer", req, &resp); err != nil {
				return err
			}

			deployer := getDeployerInfo(resp.Deployer)

			output, err := json.MarshalIndent(deployer, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const listDeployersCmdExample = `
loom deployer list
`

func listDeployersCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "Display all deployers in whitelist",
		Example: listDeployersCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			req := &dwtypes.ListDeployersRequest{}
			var resp dwtypes.ListDeployersResponse
			if err := cli.StaticCallContractWithFlags(&flags, dwContractName, "ListDeployers", req, &resp); err != nil {
				return err
			}

			deployersInfo := []*deployerInfo{}
			for _, deployer := range resp.Deployers {
				deployerInfo := getDeployerInfo(deployer)
				deployersInfo = append(deployersInfo, &deployerInfo)
			}

			output, err := json.MarshalIndent(deployersInfo, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

func getDeployerInfo(deployer *dwtypes.Deployer) deployerInfo {
	flagsInt := dw.UnpackFlags(deployer.Flags)
	flags := []string{}
	for _, flag := range flagsInt {
		flags = append(flags, dwtypes.Flags_name[int32(flag)])
	}
	f := strings.Join(flags, "|")
	deployerInfo := deployerInfo{
		Address: deployer.Address.ChainId + ":" + deployer.Address.Local.String(),
		Flags:   f,
	}
	return deployerInfo
}
