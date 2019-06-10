package userdeployerwhitelist

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/loomnetwork/go-loom/client"

	"github.com/pkg/errors"

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
loom dev add-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857 --tier 0 
`

func addUserDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var tierID int
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
			req := &udwtypes.WhitelistUserDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
				TierID:       udwtypes.TierID(tierID),
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "AddUserDeployer", req, nil)
		},
	}
	cmd.Flags().IntVarP(&tierID, "tier", "t", 0, "tier ID")
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
loom dev get-tier 0
`

func getTierInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-tier <tier>",
		Short:   "Show tier details",
		Example: getTierCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tierID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "tierID %s does not parse as integer", args[0])
			}
			req := &udwtypes.GetTierInfoRequest{
				TierID: udwtypes.TierID(tierID),
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
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const SetTierCmdExample = `
loom dev set-tier 0 --fee 100 --tierName Tier1 
`

func setTierInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var inputFee, tierName string
	cmd := &cobra.Command{
		Use:     "set-tier <tier>",
		Short:   "Set tier details",
		Example: SetTierCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var fee *types.BigUInt
			tierID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "tierID %s does not parse as integer", args[0])
			}
			rpcClient := getDAppChainClient(&flags)
			udwAddress, err := rpcClient.Resolve("user-deployer-whitelist")
			if err != nil {
				return errors.Wrap(err, "failed to resolve user-deployer-whitelist address")
			}
			udwContract := client.NewContract(rpcClient, udwAddress.Local)
			getTierInfoReq := &udwtypes.GetTierInfoRequest{
				TierID: udwtypes.TierID(tierID),
			}
			var getTierInfoResp udwtypes.GetTierInfoResponse
			_, err = udwContract.StaticCall("GetTierInfo", getTierInfoReq, udwAddress, &getTierInfoResp)
			if err != nil {
				return errors.Wrap(err, "failed to call GetTierInfo")
			}
			if len(inputFee) == 0 {
				fee = getTierInfoResp.Tier.Fee
			} else {
				parsedFee, err := cli.ParseAmount(inputFee)
				if err != nil {
					return err
				}
				if parsedFee.Cmp(loom.NewBigUIntFromInt(0)) <= 0 {
					return fmt.Errorf("fee must be greater than zero")
				}
				fee = &types.BigUInt{
					Value: *parsedFee,
				}
			}
			if len(tierName) == 0 {
				tierName = getTierInfoResp.Tier.Name
			}
			req := &udwtypes.SetTierInfoRequest{
				Fee:    fee,
				Name:   tierName,
				TierID: udwtypes.TierID(tierID),
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "SetTierInfo", req, nil)
		}}

	cmd.Flags().StringVarP(&inputFee, "fees", "f", "", "tier ID")
	cmd.Flags().StringVarP(&tierName, "tierName", "t", "", "tier ID")
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

func getDAppChainClient(callFlags *cli.ContractCallFlags) *client.DAppChainRPCClient {
	writeURI := callFlags.URI + "/rpc"
	readURI := callFlags.URI + "/query"
	return client.NewDAppChainRPCClient(callFlags.ChainID, writeURI, readURI)
}
