package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/loomnetwork/loomchain/integration-test/engine"
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

			// tc, err := lib.ReadTestCases(testFilename)
			// if err != nil {
			// 	return err
			// }

			var testcases = []lib.TestCase{
				lib.TestCase{
					RunCmd:    fmt.Sprintf(`example-cli call balance {{index $.AccountAddressList 0}}`),
					Condition: "contains",
					Expected:  "100000000000000000000",
				},
				lib.TestCase{
					RunCmd:    fmt.Sprintf(`example-cli call balance {{index $.AccountAddressList 1}}`),
					Condition: "contains",
					Expected:  "100000000000000000000",
				},
				lib.TestCase{
					RunCmd:    fmt.Sprintf(`example-cli call balance {{index $.AccountAddressList 2}}`),
					Condition: "contains",
					Expected:  "100000000000000000000",
				},
			}

			// Trap Interrupts, SIGINTs and SIGTERMs.
			sigC := make(chan os.Signal, 1)
			signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigC)

			errC := make(chan error)
			e := engine.NewCmd(conf, lib.TestCases(testcases))

			ctx, cancel := context.WithCancel(context.Background())
			go func() { errC <- e.Run(ctx) }()

			err = func() error {
				for {
					select {
					case err := <-errC:
						cancel()
						return err
					case <-sigC:
						cancel()
						fmt.Printf("stopping runner\n")
						return nil
					}
				}
			}()

			return err
		},
	}

	flags := command.Flags()
	flags.StringVar(&confFilename, "conf", "default/runner.toml", "Runner configuration path")
	flags.StringVar(&testFilename, "test", "test.toml", "Test file path")
	return command
}
