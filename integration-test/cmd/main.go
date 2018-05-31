package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dpos",
		Short: "dpos integration test",
	}

	rootCmd.AddCommand(newNewCommand())
	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newTestCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
