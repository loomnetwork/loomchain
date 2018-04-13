//+build !BUILTIN_PLUGINS

package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"plugin"

	lp "github.com/loomnetwork/loom-plugin"
)

var (
	errPluginSymbolWrongType = errors.New("CmdPlugin symbol has wrong type")
)

type PluginEntry struct {
	lp.CmdPlugin
	Path string
}

type CmdPluginManager struct {
	RootCmd *lp.Command
	lp.CmdPluginSystem
	Dir string
}

func (m *CmdPluginManager) List() ([]*PluginEntry, error) {
	files, err := ioutil.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}

	var entries []*PluginEntry
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fullPath := path.Join(m.Dir, file.Name())
		cmdPlugin, err := loadPlugin(fullPath)
		if err != nil {
			fmt.Printf("encountered invalid plugin at %s\n%s\n", fullPath, err.Error())
			continue
		}

		entries = append(entries, &PluginEntry{
			Path:      fullPath,
			CmdPlugin: cmdPlugin,
		})
	}

	return entries, nil
}

func (m *CmdPluginManager) ActivatePlugin(cmdPlugin lp.CmdPlugin) error {
	if err := cmdPlugin.Init(m.CmdPluginSystem); err != nil {
		return err
	}
	m.RootCmd.AddCommand(cmdPlugin.GetCmds()...)
	return nil
}

func loadPlugin(path string) (lp.CmdPlugin, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	symbol, err := plug.Lookup("CmdPlugin")
	if err != nil {
		return nil, err
	}

	cmdPlugin, ok := symbol.(lp.CmdPlugin)
	if !ok {
		return nil, errPluginSymbolWrongType
	}

	return cmdPlugin, nil
}
