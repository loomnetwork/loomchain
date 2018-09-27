package main

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "loomdb-viewer",
		Short: "tool for viewing loom db",
	}
	
	rootCmd.AddCommand(loadCmd())
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func loadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invstigate [name] [path]",
		Short: "call a contract method",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return investigate(args[0], args[1])
		},
	}
	return cmd
}

