//+build BUILTIN_PLUGINS

package cli

import (
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/spf13/cobra"
)

type BuiltinCmdPluginManager struct {
	RootCmd *cobra.Command
	lp.CmdPluginSystem
}

func (m *BuiltinCmdPluginManager) ActivatePlugin(cmdPlugin lp.CmdPlugin) error {
	if err := cmdPlugin.Init(m.CmdPluginSystem); err != nil {
		return err
	}
	m.RootCmd.AddCommand(cmdPlugin.GetCmds()...)
	return nil
}
