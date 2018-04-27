//+build BUILTIN_PLUGINS

package main

import (
	"fmt"
	"os"

	lp "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/examples/cmd-plugins/create-tx/plugin"
	"github.com/loomnetwork/loom/cli"
)

// rootCmd is the entry point for this binary
var rootCmd = &lp.Command{
	Use:   "ladmin",
	Short: "Loom Admin CLI (Debug Mode)",
}

func main() {
	// NOTE: The VSCode Go plugin will indicate that BuiltinCmdPluginManager is undefined
	//       unless go.buildTags contains BUILTIN_PLUGINS, can just ignore that as long as
	//       the build tag is actually set when the executable is built.
	pm := cli.BuiltinCmdPluginManager{
		RootCmd:         rootCmd,
		CmdPluginSystem: cli.NewCmdPluginSystem(),
	}
	// activate built-in cmd plugins
	createTxCmd := &plugin.CreateTxCmdPlugin{}
	pm.ActivatePlugin(createTxCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
