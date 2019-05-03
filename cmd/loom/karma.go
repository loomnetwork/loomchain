package main

import (
	"fmt"
	"strconv"

	"github.com/loomnetwork/loomchain/cmd/loom/common"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	KarmaContractName = "karma"
)

func GetSourceCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-sources",
		Short: "list the karma sources",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp ktypes.KarmaSources
			err := cli.StaticCallContractWithFlags(flags, KarmaContractName, "GetSources", &ktypes.GetSourceRequest{}, &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func GetUserStateCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-user-state <user> <address>",
		Short: "list the karma sources for user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			var resp ktypes.KarmaState
			err = cli.StaticCallContractWithFlags(flags, KarmaContractName, "GetUserState", addr.MarshalPB(), &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func GetUserTotalCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-total <user> <target>",
		Short: "Check amount of karma user has, target can be either CALL or DEPLOY",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			userTarget := ktypes.KarmaUserTarget{
				User: addr.MarshalPB(),
			}

			userTarget.Target, err = readTarget(args[1])
			if err != nil {
				return err
			}

			var resp ktypes.KarmaTotal
			err = cli.StaticCallContractWithFlags(flags, KarmaContractName, "GetUserKarma", &userTarget, &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func DepositCoinCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "deposit-coin <user> <amount>",
		Short: "deposit coin for deploys to the user's karma",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return errors.Wrap(err, "parse amount arg")
			}

			depositAmount := ktypes.KarmaUserAmount{
				User:   user.MarshalPB(),
				Amount: &types.BigUInt{Value: *amount},
			}

			err = cli.CallContractWithFlags(flags, KarmaContractName, "DepositCoin", &depositAmount, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("coin successfully deposited")
			return nil
		},
	}
}

func WithdrawCoinCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "withdraw-coin <user> <amount>",
		Short: "withdraw coin for deploys to the user's karma",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return errors.Wrap(err, "parse amount arg")
			}

			withdrawAmount := ktypes.KarmaUserAmount{
				User:   user.MarshalPB(),
				Amount: &types.BigUInt{Value: *amount},
			}

			err = cli.CallContractWithFlags(flags, KarmaContractName, "WithdrawCoin", &withdrawAmount, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("coin successfully withdrawn")
			return nil
		},
	}
}

func GetConfigCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-config",
		Short: "list the karma configuration settings",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp ktypes.KarmaConfig
			err := cli.StaticCallContractWithFlags(flags, KarmaContractName, "GetConfig", &ktypes.GetConfigRequest{}, &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func SetConfigCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-config <min-karma-to-deploy>",
		Short: "set the karma configuration settings",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "parse amount as integer %v", args[0])
			}
			err = cli.CallContractWithFlags(flags, KarmaContractName, "SetConfig", &ktypes.KarmaConfig{
				MinKarmaToDeploy: amount,
			}, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("config successfully updated")
			return nil
		},
	}
}

func AddKarmaCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add-karma <user> [ (source, count) ]...",
		Short: "add new source of karma to a user, requires oracle verification",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := cli.ParseAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			req := ktypes.AddKarmaRequest{
				User: user.MarshalPB(),
			}

			if len(args)%2 != 1 {
				return errors.New("incorrect argument count, should be odd")
			}
			numNewSources := (len(args) - 1) / 2
			for i := 0; i < numNewSources; i++ {
				count, err := strconv.ParseInt(args[2*i+2], 10, 64)
				if err != nil {
					return errors.Wrapf(err, "cannot convert %s to integer", args[2*i+2])
				}
				req.KarmaSources = append(req.KarmaSources, &ktypes.KarmaSource{
					Name:  args[2*i+1],
					Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(count)},
				})
			}

			err = cli.CallContractWithFlags(flags, KarmaContractName, "AddKarma", &req, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("user's sources successfully updated")
			return nil
		},
	}
}

func SetActiveCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-active <contract>",
		Short: "set contract as active",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contract, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			err = cli.CallContractWithFlags(flags, KarmaContractName, "SetActive", contract.MarshalPB(), nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("contract", contract.String(), "set active")
			return nil
		},
	}
}

func SetInactiveCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-inactive <contract>",
		Short: "set contract as inactive",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contract, err := cli.ResolveAddress(args[0], cli.TxFlags.ChainID, cli.TxFlags.URI)
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			err = cli.CallContractWithFlags(flags, KarmaContractName, "SetInactive", contract.MarshalPB(), nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("contract", contract.String(), "set inactive")
			return nil
		},
	}
}

func SetUpkeepCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-upkeep <cost> <period>",
		Short: "set upkeep parameters",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cost, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "cost %s does not parse as integer", args[0])
			}
			period, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "cost %s does not parse as integer", args[2])
			}
			err = cli.CallContractWithFlags(flags, KarmaContractName, "SetUpkeepParams", &ktypes.KarmaUpkeepParams{
				Cost:   cost,
				Period: period,
			}, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("upkeep parameters updated")
			return nil
		},
	}
}

func GetUpkeepCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-upkeep",
		Short: "get upkeep parameters",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp ktypes.KarmaUpkeepParams
			err := cli.StaticCallContractWithFlags(flags, KarmaContractName, "GetUpkeepParms", &types.Address{}, &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func DeleteSourcesForUserCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-sources <user> [name]...",
		Short: "Delete one or more Karma sources for a user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := cli.ParseAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			deletedStates := ktypes.KarmaStateKeyUser{
				User: user.MarshalPB(),
			}
			for i := 1; i < len(args); i++ {
				deletedStates.StateKeys = append(deletedStates.StateKeys, args[i])
			}

			err = cli.CallContractWithFlags(flags, KarmaContractName, "DeleteSourcesForUser", &deletedStates, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("sources successfully deleted")
			return nil
		},
	}
}

func ResetSourcesCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "reset-sources [ (source reward target) ]...",
		Short: "reset the sources, requires oracle verification",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var newSources ktypes.KarmaSources
			if len(args)%3 != 0 {
				return errors.New("incorrect argument count, should be multiple of three")
			}
			numNewSources := len(args) / 3
			for i := 0; i < numNewSources; i++ {
				reward, err := strconv.ParseInt(args[3*i+1], 10, 64)
				if err != nil {
					return errors.Wrapf(err, "cannot convert %s to integer", args[3*i+1])
				}
				target, err := readTarget(args[3*i+2])
				if err != nil {
					return err
				}

				newSources.Sources = append(newSources.Sources, &ktypes.KarmaSourceReward{
					Name:   args[3*i],
					Reward: reward,
					Target: target,
				})
			}

			err := cli.CallContractWithFlags(flags, KarmaContractName, "ResetSources", &newSources, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("sources successfully updated")
			return nil
		},
	}
}

func readTarget(target string) (ktypes.KarmaSourceTarget, error) {
	if value, ok := ktypes.KarmaSourceTarget_value[target]; ok {
		return ktypes.KarmaSourceTarget(value), nil
	}

	targetValue, err := strconv.ParseInt(target, 10, 32)
	if err != nil {
		return 0, errors.Errorf("unrecognised input karma source target %s", target)
	}
	t := ktypes.KarmaSourceTarget(targetValue)
	if t == ktypes.KarmaSourceTarget_CALL || t == ktypes.KarmaSourceTarget_DEPLOY {
		return t, nil
	}
	return 0, errors.Errorf("unrecognised karma source target %s", target)

}

func UpdateOracleCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "update-oracle <new-oracle>",
		Short: "change the oracle or set initial oracle",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newOracle, err := cli.ParseAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve new oracle address arg")
			}

			err = cli.CallContractWithFlags(flags, KarmaContractName, "UpdateOracle", &ktypes.KarmaNewOracle{
				NewOracle: newOracle.MarshalPB(),
			}, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("oracle changed")
			return nil
		},
	}
}

func AddKarmaMethods() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "karma <command>",
		Short: "Methods available in karma contract",
	}
	var flags, staticflags cli.ContractCallFlags
	common.AddContractCallFlags(cmd.PersistentFlags(), &flags)
	common.AddContractStaticCallFlags(cmd.PersistentFlags(), &staticflags)
	cmd.AddCommand(
		GetSourceCmd(&staticflags),
		GetUserStateCmd(&staticflags),
		GetUserTotalCmd(&staticflags),
		DepositCoinCmd(&flags),
		WithdrawCoinCmd(&flags),
		AddKarmaCmd(&flags),
		SetActiveCmd(&flags),
		SetInactiveCmd(&flags),
		SetUpkeepCmd(&flags),
		GetUpkeepCmd(&staticflags),
		GetConfigCmd(&staticflags),
		SetConfigCmd(&flags),
		DeleteSourcesForUserCmd(&flags),
		ResetSourcesCmd(&flags),
		UpdateOracleCmd(&flags),
	)
	return cmd
}
