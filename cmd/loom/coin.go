package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
)

const CoinContractName = "coin"

func TransferCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "transfer [address] [amount]",
		Short: "Transfer coins to another account",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}

			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}
			return cli.CallContractWithFlags(&flags, CoinContractName, "Transfer", &coin.TransferRequest{
				To: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func TransferFromCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "transfer_from [from_address] [to_address] [amount]",
		Short: "Transfer coins from a specified address to another",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromAddress, err := cli.ResolveAddress(args[0], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}
			toAddress, err := cli.ResolveAddress(args[1], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[2])
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(&flags, CoinContractName, "TransferFrom", &coin.TransferFromRequest{
				From: fromAddress.MarshalPB(),
				To:   toAddress.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd

}

func ApproveCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "approve [address] [amount]",
		Short: "Approve the transfer of coins to another account",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(&flags, CoinContractName, "Approve", &coin.ApproveRequest{
				Spender: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}

	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func AllowanceCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "allowance [owner] [spender]",
		Short: "Check the pre-approved amount that the spender can transfer from the owner account",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ownerAddr, err := cli.ResolveAddress(args[0], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}

			spenderAddr, err := cli.ResolveAddress(args[1], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}

			var resp coin.AllowanceResponse
			err = cli.StaticCallContractWithFlags(&flags, CoinContractName, "Allowance", &coin.AllowanceRequest{
				Owner:   ownerAddr.MarshalPB(),
				Spender: spenderAddr.MarshalPB(),
			}, &resp)
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

	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

func BalanceCmd() *cobra.Command {
	var staticflags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "balance [address]",
		Short: "Fetch the balance of a coin account",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], staticflags.ChainID, staticflags.URI)
			if err != nil {
				return err
			}
			var resp coin.BalanceOfResponse
			err = cli.StaticCallContractWithFlags(&staticflags, CoinContractName, "BalanceOf", &coin.BalanceOfRequest{
				Owner: addr.MarshalPB(),
			}, &resp)
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &staticflags)
	return cmd
}

func BalancesCmd() *cobra.Command {
	var staticflags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "balances",
		Short: "Fetch the balances of all coin accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp coin.BalancesResponse
			if err := cli.StaticCallContractWithFlags(
				&staticflags, CoinContractName, "Balances", &coin.BalancesRequest{}, &resp,
			); err != nil {
				return err
			}
			type accountBalance struct {
				Owner   string `json:"owner"`
				Balance string `json:"balance"`
			}
			balances := []accountBalance{}
			for _, a := range resp.Accounts {
				addr := loom.UnmarshalAddressPB(a.Owner)
				balance := a.Balance.Value.Int.String()
				balances = append(balances, accountBalance{
					Owner:   addr.String(),
					Balance: balance,
				})
			}
			prettyJSON, err := json.MarshalIndent(balances, "", "    ")
			if err != nil {
				return errors.Wrap(err, "failed to generate json output")
			}
			fmt.Println(string(prettyJSON))
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &staticflags)
	return cmd
}

func NewCoinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coin <command>",
		Short: "Methods available in coin contract",
	}

	cmd.AddCommand(
		ApproveCmd(),
		AllowanceCmd(),
		BalanceCmd(),
		TransferCmd(),
		TransferFromCmd(),
		BalancesCmd(),
	)
	return cmd
}
