package main

import (
	"errors"

	"github.com/loomnetwork/loomchain/gen"
	"github.com/spf13/cobra"
)

var defaultProjectName = ""

type spinFlags struct {
	OutDir string `json:"outDir"`
	Name   string `json:"name"`
}

func newSpinCommand() *cobra.Command {
	var flags spinFlags

	spinCmd := &cobra.Command{
		Use:   "spin",
		Short: "Spin a loom project from a github repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return errors.New("No repository entered")
			}

			return gen.Spin(args[0], flags.OutDir, flags.Name)
		},
	}
	spinCmd.Flags().StringVar(&flags.OutDir, "outDir", "", "output directory")
	spinCmd.Flags().StringVar(&flags.Name, "name", defaultProjectName, "project name")
	return spinCmd

}
