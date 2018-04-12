package cli

type CmdPlugin interface {
	Init(sys CmdPluginSystem) error
	GetCmds() []*Cmd
}
