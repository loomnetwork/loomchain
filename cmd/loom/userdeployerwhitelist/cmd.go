package userdeployerwhitelist

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/loomnetwork/go-loom"

	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
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
		removeUserDeployerCmd(),
		swapUserDeployerCmd(),
		getUserDeployersCmd(),
		getDeployedContractsCmd(),
		getTierInfoCmd(),
		setTierInfoCmd(),
		destroyDeployedContractCmd(),
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

const removeUserDeployerCmdExample = `
loom dev remove-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func removeUserDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use: "remove-deployer <deployer address>",
		// nolint:lll
		Short:   "Remove an account from the list of accounts authorized to deploy contracts on behalf of a user (the caller)",
		Example: removeUserDeployerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			req := &udwtypes.RemoveUserDeployerRequest{
				DeployerAddr: addr.MarshalPB(),
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "RemoveUserDeployer", req, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const swapUserDeployerCmdExample = `
loom dev swap-deployer 0x7262d4c97c7B93937E4810D289b7320e9dA82857 0x7262d4c97c7B93937E4810D289b7320e9dA82859
`

func swapUserDeployerCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use: "swap-deployer <old-deployer address> <new-deployer address>",
		// nolint:lll
		Short:   "Swap an account from the list of accounts authorized to deploy contracts on behalf of a user to new account (the caller)",
		Example: swapUserDeployerCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldDeployerAddr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			newDeployerAddr, err := cli.ResolveAccountAddress(args[1], &flags)
			if err != nil {
				return err
			}
			req := &udwtypes.SwapUserDeployerRequest{
				OldDeployerAddr: oldDeployerAddr.MarshalPB(),
				NewDeployerAddr: newDeployerAddr.MarshalPB(),
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "SwapUserDeployer", req, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const destroyDeployedContractCmdExample = `
loom dev destroy-contract 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func destroyDeployedContractCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use: "destroy-contract <contract address>",
		// nolint:lll
		Short:   "Destroy a deployed contract by contract address",
		Example: destroyDeployedContractCmdExample,
		Args:    cobra.RangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contractAddr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			fmt.Println("contractAddress", contractAddr.String())
			req := &udwtypes.DestroyDeployedContractsRequest{
				ContractAddress: contractAddr.MarshalPB(),
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "DestroyDeployedContract", req, nil)
		},
	}
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
			err = cli.StaticCallContractWithFlags(&flags, dwContractName, "GetTierInfo", req, &resp)
			if err != nil {
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

const setTierCmdExample = `
loom dev set-tier 0 --fee 100 --name Tier1 --block-range 10 --max-txs 2 
`

func setTierInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var inputFee, tierName string
	var blockRange, maxTxs uint64
	cmd := &cobra.Command{
		Use:     "set-tier <tier> [options]",
		Short:   "Set tier details",
		Example: setTierCmdExample,
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
			if blockRange == 0 {
				blockRange = getTierInfoResp.Tier.BlockRange
			}
			if blockRange == 0 {
				return fmt.Errorf("block range must be greater than zero")
			}
			if maxTxs == 0 {
				maxTxs = getTierInfoResp.Tier.MaxTxs
			}
			if maxTxs == 0 {
				return fmt.Errorf("max-txs must be greater than zero")
			}
			req := &udwtypes.SetTierInfoRequest{
				Fee:        fee,
				Name:       tierName,
				TierID:     udwtypes.TierID(tierID),
				BlockRange: blockRange,
				MaxTxs:     maxTxs,
			}
			return cli.CallContractWithFlags(&flags, dwContractName, "SetTierInfo", req, nil)
		}}

	cmd.Flags().StringVarP(&inputFee, "fee", "f", "", "Tier fee")
	cmd.Flags().StringVarP(&tierName, "name", "n", "", "Tier name")
	cmd.Flags().Uint64Var(&blockRange, "block-range", 0, "Block range")
	cmd.Flags().Uint64Var(&maxTxs, "max-txs", 0, "Max txs per block range")
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
