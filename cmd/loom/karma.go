package main

import (
	"fmt"
	"strconv"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	KarmaContractName = "karma"
)

func GetSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-sources",
		Short: "list the karma sources",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp ktypes.KarmaSources
			err := staticCallContract(KarmaContractName, "GetSources", &types.Address{}, &resp)
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

func GetUserStateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-user-state (user address)",
		Short: "list the karma sources for user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			var resp ktypes.KarmaState
			err = staticCallContract(KarmaContractName, "GetUserState", addr.MarshalPB(), &resp)
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

func GetUserTotalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-total (user address)",
		Short: "calculate total karma for user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			var resp ktypes.KarmaTotal
			err = staticCallContract(KarmaContractName, "GetTotal", addr.MarshalPB(), &resp)
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

func DepositCoinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deposit-coin (user amount)",
		Short: "deposit coin for deploys to the user's karma",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			amount, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return errors.Wrap(err, "parse amount arg")
			}

			depositAmount := ktypes.KarmaUserAmount{
				User:   user.MarshalPB(),
				Amount: &types.BigUInt{*loom.NewBigUIntFromInt(amount)},
			}

			err = callContract(KarmaContractName, "DepositCoin", &depositAmount, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("coin successfully deposited")
			return nil
		},
	}
}

func WithdrawCoinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "withdraw-coin (user amount)",
		Short: "withdraw coin for deploys to the user's karma",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			amount, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return errors.Wrap(err, "parse amount arg")
			}

			withdrawAmount := ktypes.KarmaUserAmount{
				User:   user.MarshalPB(),
				Amount: &types.BigUInt{*loom.NewBigUIntFromInt(amount)},
			}

			err = callContract(KarmaContractName, "WithdrawCoin", &withdrawAmount, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("coin successfully withdrawn")
			return nil
		},
	}
}

func AppendSourcesForUserCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "append-sources-for-user (user) (oracle) [ [source] [count] ]...",
		Short: "add new source of karma to a user, requires oracle verification",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			oracle, err := resolveAddress(args[1])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			newStateUser := ktypes.KarmaStateUser{
				User:   user.MarshalPB(),
				Oracle: oracle.MarshalPB(),
			}
			if len(args)%2 != 0 {
				return errors.New("incorrect argument count, should be even")
			}
			numNewSources := (len(args) - 2) / 2
			for i := 0; i < numNewSources; i++ {
				count, err := strconv.ParseInt(args[2*i+3], 10, 64)
				if err != nil {
					return errors.Wrapf(err, "cannot convert %s to integer", args[2*i+3])
				}
				newStateUser.SourceStates = append(newStateUser.SourceStates, &ktypes.KarmaSource{
					Name:  args[2*i+2],
					Count: count,
				})
			}

			err = callContract(KarmaContractName, "AppendSourcesForUser", &newStateUser, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("user's sources successfully updated")
			return nil
		},
	}
}

func DeleteSourcesForUserCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-sources-for-user (user) (oracle) [name]...",
		Short: "delete sources assigned to user, requires oracle verification",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			oracle, err := resolveAddress(args[1])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			deletedStates := ktypes.KarmaStateKeyUser{
				User:   user.MarshalPB(),
				Oracle: oracle.MarshalPB(),
			}
			for i := 2; i < len(args); i++ {
				deletedStates.StateKeys = append(deletedStates.StateKeys, args[i])
			}

			err = callContract(KarmaContractName, "DeleteSourcesForUser", &deletedStates, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("sources successfully deleted")
			return nil
		},
	}
}

func ResetSourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-sources (oracle) [ [source] [reward] [target] ]...",
		Short: "reset the sources, requires oracle verification",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oracle, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}

			newSources := ktypes.KarmaSourcesValidator{
				Oracle: oracle.MarshalPB(),
			}
			a := len(args)%3; a=a
			if len(args)%3 != 1 {
				return errors.New("incorrect argument count, should be odd")
			}
			numNewSources := (len(args) - 1) / 3
			for i := 0; i < numNewSources; i++ {
				reward, err := strconv.ParseInt(args[3*i+2], 10, 64)
				if err != nil {
					return errors.Wrapf(err, "cannot convert %s to integer", args[2*i+2])
				}
				target,err := readTarget(args[3*i + 3])
				if err != nil {
					return err
				}

				newSources.Sources = append(newSources.Sources, &ktypes.KarmaSourceReward{
					Name:   args[3*i+1],
					Reward: reward,
					Target: target,
				})
			}

			err = callContract(KarmaContractName, "ResetSources", &newSources, nil)
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
		return 0, errors.Errorf( "unrecognised input karma source target %s", target)
	}
	t := ktypes.KarmaSourceTarget(targetValue)
	if t == ktypes.KarmaSourceTarget_CALL || t == ktypes.KarmaSourceTarget_ALL || t == ktypes.KarmaSourceTarget_DEPLOY {
		return t, nil
	}
	return 0, errors.Errorf("unrecognised karma source target %s", target)

}

func UpdateOracleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-oracle (new oracle) [old oracle]",
		Short: "change the oracle or set initial oracle",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newOracle, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve new oracle address arg")
			}
			var oldOracle loom.Address
			if len(args) > 1 {
				oldOracle, err = resolveAddress(args[1])
				if err != nil {
					return errors.Wrap(err, "resolve old oracle address arg")
				}
			}

			err = callContract(KarmaContractName, "UpdateOracle", &ktypes.KarmaNewOracleValidator{
				NewOracle: newOracle.MarshalPB(),
				OldOracle: oldOracle.MarshalPB(),
			}, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("oracle changed")
			return nil
		},
	}
}

func AddKarmaMethods(karmaCmd *cobra.Command) {
	karmaCmd.AddCommand(
		GetSourceCmd(),
		GetUserStateCmd(),
		GetUserTotalCmd(),
		DepositCoinCmd(),
		WithdrawCoinCmd(),
		AppendSourcesForUserCmd(),
		DeleteSourcesForUserCmd(),
		ResetSourcesCmd(),
		UpdateOracleCmd(),
	)
}
