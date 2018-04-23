package main

import (
	"errors"

	"github.com/loomnetwork/loom/gen"
	"github.com/spf13/cobra"
)

var defaultProjectName = "MyLoomProject"

type unboxFlags struct {
	OutDir string 	`json:"outDir"`
	Name string 	`json:"name"`
}
func newUnboxCommand() *cobra.Command {
	var flags unboxFlags

	unboxCmd := &cobra.Command{
		Use:   "unbox",
		Short: "Unbox a loom project from a github repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return errors.New("No repository entered")
			}

			return gen.Unbox(args[0], flags.OutDir, flags.Name)
		},
	}
	unboxCmd.Flags().StringVar(&flags.OutDir, "outDir", "", "output directory")
	unboxCmd.Flags().StringVar(&flags.Name, "name", defaultProjectName, "project name")
	return unboxCmd

}





















