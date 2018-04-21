package main

import (
	"errors"
	"github.com/spf13/cobra"
	"os/exec"
	"os"
	"path/filepath"
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
			return unbox(getRepoPath(args[0]), getOutDir(flags))
		},
	}
	unboxCmd.Flags().StringVar(&flags.OutDir, "outDir", "", "output directory")
	unboxCmd.Flags().StringVar(&flags.Name, "name", defaultProjectName, "project name")
	return unboxCmd

}

func unbox(gitRepo string, outdir string) error {
	cloneCmd := exec.Command("git","clone", gitRepo, outdir)
	err := cloneCmd.Run()
	return err
}

func getRepoPath(args0 string) (string) {
	var gitRepo string
	if len(args0) >= 19 && args0[0:19] == "https://github.com/" {
		gitRepo = args0
	} else if len(args0) >= 11 && args0[0:11] == "github.com/" {
		gitRepo = "https://" + args0
	} else {
		gitRepo = "https://github.com/loomnetwork/" + args0
	}
	return gitRepo
}

func getOutDir(flags unboxFlags) (string) {
	if len(flags.OutDir) == 0 {
		outdir := filepath.Join(os.Getenv("GOPATH"),"src","github.com",os.Getenv("USER"),flags.Name)
		return outdir
	} else {
		return filepath.Join(flags.OutDir, flags.Name)
	}
	
}