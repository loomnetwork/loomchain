//+build BUILTIN_PLUGINS

package cli

type BuiltinCmdPluginManager struct {
	RootCmd *Cmd
	CmdPluginSystem
}

func (m *BuiltinCmdPluginManager) ActivatePlugin(cmdPlugin CmdPlugin) error {
	if err := cmdPlugin.Init(m.CmdPluginSystem); err != nil {
		return err
	}
	m.RootCmd.AddCommand(cmdPlugin.GetCmds()...)
	return nil
}
