package main

import (
	"fmt"
	"os"
	"path/filepath"

	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/cli"
	"github.com/spf13/viper"
)

// rootCmd is the entry point for this binary
var rootCmd = &lp.Command{
	Use:   "ladmin",
	Short: "Loom Admin CLI",
}

const (
	cmdPluginDirKey = "CmdPluginDir"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// load & activate any cmd plugins in the out/cmds dir
	pluginsDir := filepath.Join(filepath.Dir(exe), "out/cmds")

	viper.SetEnvPrefix("LOOM")
	viper.SetDefault(cmdPluginDirKey, pluginsDir)
	viper.BindEnv(cmdPluginDirKey)
	pluginsDir = viper.GetString(cmdPluginDirKey)

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
