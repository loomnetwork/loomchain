package main

import (
	`fmt`
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/types"
	`github.com/pkg/errors`
	`github.com/spf13/cobra`
)

const (
	KarmaContractName = "karma"
)

func GetSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-source",
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
				return errors.Wrap(err,"formt JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}
// func (k *Karma) GetState(ctx contract.StaticContext, user *types.Address) (*State, error) {
func GetUserStateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-user-state [user address]",
		Short: "list the karma sources for user",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := resolveAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "resolve address arg")
			}
			
			var resp ktypes.KarmaState
			err = staticCallContract(KarmaContractName, "GetSources", addr.MarshalPB(), &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err,"formt JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func AddKarmaMethods(karmaCmd *cobra.Command) {
	karmaCmd.AddCommand(
		GetSourceCmd(),
		GetUserStateCmd(),
	)
}