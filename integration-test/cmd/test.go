package main

import (
	"fmt"

	"github.com/loomnetwork/loomchain/integration-test/lib"
	"github.com/spf13/cobra"
)

func newTestCommand() *cobra.Command {
	var confFilename, testFilename string
	command := &cobra.Command{
		Use:           "test",
		Short:         "Test nodes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(ccmd *cobra.Command, args []string) error {
			conf, err := lib.ReadConfig(confFilename)
			if err != nil {
				return err
			}
			fmt.Printf("%#v\n", conf)

			// tc, err := lib.ReadTestCases(testFilename)
			// if err != nil {
			// 	return err
			// }

			return nil
		},
	}

	flags := command.Flags()
	flags.StringVar(&confFilename, "conf", "default/runner.toml", "Runner configuration path")
	flags.StringVar(&testFilename, "test", "test.toml", "Test file path")
	return command
}
