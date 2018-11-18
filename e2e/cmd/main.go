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

	rootCmd.AddCommand(
		newNewCommand(),
		newRunCommand(),
		newTestCommand(),
		newGenerateCommand(),
		newPubKeyCommand(),
		newImportCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
