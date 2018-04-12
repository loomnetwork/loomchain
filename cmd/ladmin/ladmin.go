package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loomnetwork/loom/cli"
	"github.com/spf13/cobra"
)

// rootCmd is the entry point for this binary
var rootCmd = &cobra.Command{
	Use:   "ladmin",
	Short: "Loom Admin CLI",
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// load & activate any cmd plugins in the out/cmds dir
	pluginsDir := filepath.Join(filepath.Dir(exe), "out/cmds")
	fmt.Printf("loading cmd plugins from %s\n", pluginsDir)
	pm := cli.CmdPluginManager{
		RootCmd:         rootCmd,
		CmdPluginSystem: cli.NewCmdPluginSystem(),
		Dir:             pluginsDir,
	}
	plugins, err := pm.List()
	if err != nil {
		panic(err)
	}
	for _, p := range plugins {
		pm.ActivatePlugin(p)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
