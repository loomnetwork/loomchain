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
					RunCmd: fmt.Sprintf(`example-cli call balance {{with $acct := index $.Accounts 0}}{{$acct.Address}}{{end}}`),
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

			func() {
				for {
					select {
					case err := <-errC:
						if err != nil {
							fmt.Printf("error: %#v", err)
						}
						cancel()
						return
					case <-sigC:
						cancel()
						fmt.Printf("stopping runner\n")
						return
					}
				}
			}()

			return nil
		},
	}

	flags := command.Flags()
	flags.StringVar(&confFilename, "conf", "default/runner.toml", "Runner configuration path")
	flags.StringVar(&testFilename, "test", "test.toml", "Test file path")
	return command
}
