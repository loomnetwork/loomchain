package main

import (
	"github.com/loomnetwork/loom/examples/cmd-plugins/create-tx/cmd-plugin"
)

// Create an instance of the plugin that will be loaded by the plugin manager.
var CmdPlugin cmdplugins.CreateTxCmdPlugin

// go-code-check throws up errors if this is missing...
func main() {}
