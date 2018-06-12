package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/loomnetwork/loomchain/e2e/engine"
	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
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

			fmt.Printf("reading tests from %s\n", testFilename)
			tc, err := lib.ReadTestCases(testFilename)
			if err != nil {
				return err
			}

			// Trap Interrupts, SIGINTs and SIGTERMs.
			sigC := make(chan os.Signal, 1)
			signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigC)

			errC := make(chan error)
			eventC := make(chan *node.Event)
			e := engine.NewCmd(conf, tc)

			ctx, cancel := context.WithCancel(context.Background())
			go func() { errC <- e.Run(ctx, eventC) }()

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
	flags.StringVar(&testFilename, "test", "dpos.toml", "Test file path")
	return command
}
