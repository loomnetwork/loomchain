package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "validators-tool",
		Short: "validators-tool utility",
	}

	rootCmd.AddCommand(newNewCommand())
	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newTestCommand())
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(newPubKeyCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
