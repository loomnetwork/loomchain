package userdeployerwhitelist

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"

	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/spf13/cobra"
)

var (
	dwContractName = "user-deployer-whitelist"
)

type UserdeployerInfo struct {
	Address string
	TierId  string
}

func NewUserDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev <command>",
		Short: "User Deployer Whitelist CLI",
	}
	cmd.AddCommand(
		addUserDeployerCmd(),
		getUserDeployersCmd(),
		getDeployedContractsCmd(),
		getTierInfoCmd(),
		setTierInfoCmd(),
	)
	return cmd
}

const addUserDeployerCmdExample = `
loom dev add-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857 --tier default
`

func addUserDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var TierID string
	cmd := &cobra.Command{
		Use:     "add-deployer <deployer address>",
		Short:   "Authorize an account to deploy contracts on behalf of a user (the caller)",
		Example: addUserDeployerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			var tierId udwtypes.TierID
			if strings.EqualFold(TierID, udwtypes.TierID_DEFAULT.String()) {
				tierId = udwtypes.TierID_DEFAULT
			} else {
				return fmt.Errorf("Please specify tierId <default>")
			}
			req := &udwtypes.WhitelistUserDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
				TierID:       tierId,
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "AddUserDeployer", req, nil)
		},
	}
	cmd.Flags().StringVarP(&TierID, "tier", "t", "default", "tier ID")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getUserDeployersCmdExample = `
loom dev list-deployers 0x7262d4c97c7B93937E4810D289b7320e9dA82856 
`

func getUserDeployersCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-deployers <user address>",
		Short:   "List accounts a user is allowed to deploy from",
		Example: getUserDeployersCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			req := &udwtypes.GetUserDeployersRequest{
				UserAddr: addr.MarshalPB(),
			}
			var resp udwtypes.GetUserDeployersResponse
			if err := cli.StaticCallContractWithFlags(&flags, dwContractName,
				"GetUserDeployers", req, &resp); err != nil {
				return err
			}
			deployerInfo := []UserdeployerInfo{}
			for _, deployer := range resp.Deployers {
				deployerInfo = append(deployerInfo, getUserDeployerInfo(deployer))
			}
			output, err := json.MarshalIndent(deployerInfo, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getDeployedContractsCmdExample = `
loom dev list-contracts 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func getDeployedContractsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-contracts <deployer address>",
		Short:   "List contracts deployed by a specific account",
		Example: getDeployedContractsCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			req := &udwtypes.GetDeployedContractsRequest{
				DeployerAddr: addr.MarshalPB(),
			}
			var resp udwtypes.GetDeployedContractsResponse
			if err := cli.StaticCallContractWithFlags(&flags, dwContractName,
				"GetDeployedContracts", req, &resp); err != nil {
				return err
			}
			contracts := []string{}
			for _, addr := range resp.ContractAddresses {
				contracts = append(contracts, addr.ContractAddress.ChainId+":"+addr.ContractAddress.Local.String())
			}
			output, err := json.MarshalIndent(contracts, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil

		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getTierCmdExample = `
loom dev get-tier --tier default
`

func getTierInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var TierID string
	cmd := &cobra.Command{
		Use:     "get-tier",
		Short:   "Show tier details",
		Example: getTierCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tierId udwtypes.TierID
			if strings.EqualFold(TierID, udwtypes.TierID_DEFAULT.String()) {
				tierId = udwtypes.TierID_DEFAULT
			} else {
				return fmt.Errorf("Please specify tierId <default>")
			}
			req := &udwtypes.GetTierInfoRequest{
				TierID: tierId,
			}
			var resp udwtypes.GetTierInfoResponse
			if err := cli.StaticCallContractWithFlags(&flags, dwContractName,
				"GetTierInfo", req, &resp); err != nil {
				return err
			}
			output, err := json.MarshalIndent(resp.Tier, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	cmd.Flags().StringVarP(&TierID, "tier", "t", "default", "tier ID")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const modifyTierInfoCmdExample = `
loom dev modify-tierinfo 100 Tier1 --tier default
`

func setTierInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var tierID string
	cmd := &cobra.Command{
		Use:     "set-tier <fee> <tier-name>",
		Short:   "Set tier details",
		Example: modifyTierInfoCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tierId udwtypes.TierID
			if strings.EqualFold(tierID, udwtypes.TierID_DEFAULT.String()) {
				tierId = udwtypes.TierID_DEFAULT
			} else {
				return fmt.Errorf("Please specify tierId <default>")
			}
			fees, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}
			if fees.Cmp(loom.NewBigUIntFromInt(0)) <= 0 {
				return fmt.Errorf("Whitelisting fees must be greater than zero")
			}
			req := &udwtypes.ModifyTierInfoRequest{
				Fee: &types.BigUInt{
					Value: *fees,
				},
				Name:   args[1],
				TierID: tierId,
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "ModifyTierInfo", req, nil)
		},
	}
	cmd.Flags().StringVarP(&tierID, "tier", "t", "default", "tier ID")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func getUserDeployerInfo(deployer *udwtypes.UserDeployerState) UserdeployerInfo {
	deployerInfo := UserdeployerInfo{
		Address: deployer.Address.ChainId + ":" + deployer.Address.Local.String(),
		TierId:  deployer.TierID.String(),
	}
	return deployerInfo
}
