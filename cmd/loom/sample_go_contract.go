package main

import (
	"fmt"

	"github.com/loomnetwork/go-loom/builtin/types/testing"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	SampleGoContractName = "sample-go-contract"
)

func TestingNestedEvm() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "test-nested-evm",
		Short: "test nested evm contract calls",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.CallContractWithFlags(
				&flags,
				SampleGoContractName,
				"TestNestedEvmCalls",
				&testing.TestingNestedEvmRequest{},
				nil,
			); err != nil {
				return errors.Wrap(err, "call TestingNestedEvm")
			}
			fmt.Println("Successful test nested evm call")
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func NewSampleGoContractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sample-go-contract <command>",
		Short: "Methods available in testing contract",
	}
	cmd.AddCommand(
		TestingNestedEvm(),
	)
	return cmd

}
