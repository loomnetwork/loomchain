package main

import (
	"fmt"
	"github.com/loomnetwork/loomchain/cmd/loom/common"

	"github.com/spf13/cobra"

	"github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
)

const CoinContractName = "coin"

func TransferCmd(flags *cli.ContractCallFlags) *cobra.Command {
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
			return cli.CallContractWithFlags(flags, CoinContractName, "Transfer", &coin.TransferRequest{
				To: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
	return cmd
}

func TransferFromCmd(flags *cli.ContractCallFlags) *cobra.Command {
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

			return cli.CallContractWithFlags(flags, CoinContractName, "TransferFrom", &coin.TransferFromRequest{
				From: fromAddress.MarshalPB(),
				To:   toAddress.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
	return cmd

}

func ApproveCmd(flags *cli.ContractCallFlags) *cobra.Command {
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

			return cli.CallContractWithFlags(flags, CoinContractName, "Approve", &coin.ApproveRequest{
				Spender: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
	return cmd

}

func BalanceCmd(flags *cli.ContractCallFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [address]",
		Short: "Fetch the balance of a coin account",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], flags.ChainID, flags.URI)
			if err != nil {
				return err
			}
			var resp coin.BalanceOfResponse
			err = cli.StaticCallContractWithFlags(flags, CoinContractName, "BalanceOf", &coin.BalanceOfRequest{
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
	return cmd
}

func NewCoinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coin <command>",
		Short: "Methods available in coin contract",
	}
	var flags,staticflags cli.ContractCallFlags
	common.AddContractCallFlags(cmd.PersistentFlags(), &flags)
	common.AddContractStaticCallFlags(cmd.PersistentFlags(), &staticflags)
	cmd.AddCommand(
		ApproveCmd(&flags),
		BalanceCmd(&staticflags),
		TransferCmd(&flags),
		TransferFromCmd(&flags),
	)
	return cmd
}
