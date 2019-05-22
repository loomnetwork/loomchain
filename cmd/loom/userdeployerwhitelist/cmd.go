package userdeployerwhitelist

import (
	"encoding/json"
	"fmt"
	"strings"

	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/spf13/cobra"
)

var (
	dwContractName = "user-deployer-whitelist"
)

type deployerInfo struct {
	Address string
	Flags   string
}

func NewUserDeployCommand() *cobra.Command {
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
loom userdeployer add 0x7262d4c97c7B93937E4810D289b7320e9dA82857 default
`

func addUserDeployerCmd() *cobra.Command {
	var flag cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "add <deployer address> <tierId>",
		Short:   "Add deployer corresponding to the user in tier Id <tierId> with evm permission to deployer list",
		Example: addUserDeployerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAddress(args[0])
			if err != nil {
				return err
			}
			var tierId udwtypes.TierID
			if strings.EqualFold(args[1], udwtypes.TierID_DEFAULT.String()) {
				tierId = udwtypes.TierID_DEFAULT
			} else {
				return fmt.Errorf("Please specify tierId <default>")
			}
			req := &udwtypes.WhitelistUserDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
				TierId:       tierId,
			}
			return cli.CallContractWithFlags(&flag, dwContractName, "AddUserDeployer", req, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flag)
	return cmd
}

const getUserDeployersCmdExample = `
loom userdeployer getdeployers 0x7262d4c97c7B93937E4810D289b7320e9dA82856 
`

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

func getUserDeployersCmd() *cobra.Command {
	var flag cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "getdeployers",
		Short:   "Get deployer objects corresponding to the user with EVM permision to deployer list",
		Example: getUserDeployersCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAddress(args[0])
			if err != nil {
				return err
			}
			req := &udwtypes.GetUserDeployersRequest{
				UserAddr: addr.MarshalPB(),
			}
			var resp udwtypes.GetUserDeployersResponse
			if err := cli.StaticCallContractWithFlags(&flag, dwContractName,
				"GetUserDeployers", req, &resp); err != nil {
				return err
			}
			deployerInfos := []deployerInfo{}
			for _, deployer := range resp.Deployers {
				deployerInfos = append(deployerInfos, getDeployerInfo(deployer))
			}
			output, err := json.MarshalIndent(deployerInfos, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
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
			req := &udwtypes.GetDeployedContractsRequest{
				DeployerAddr: addr.MarshalPB(),
			}
			var resp udwtypes.GetDeployedContractsResponse
			if err := cli.StaticCallContractWithFlags(&flag, dwContractName,
				"GetDeployedContracts", req, &resp); err != nil {
				return err
			}
			contracts := []string{}
			for _, addr := range resp.ContractAddresses {
				contracts = append(contracts, addr.ChainId+":"+addr.Local.String())
			}
			output, err := json.MarshalIndent(contracts, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil

		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flag)
	return cmd
}
