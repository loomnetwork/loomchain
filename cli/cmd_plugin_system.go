package cli

import (
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/client"
)

// Implements the CmdPluginSystem interface used by cmd plugins
type cmdPluginSystem struct {
}

func NewCmdPluginSystem() lp.CmdPluginSystem {
	return &cmdPluginSystem{}
}

func (ps *cmdPluginSystem) GetClient(nodeURI string) (lp.DAppChainClient, error) {
	// TODO: cache the client instead of creating a new instance every time
	return client.NewDAppChainRPCClient(nodeURI, 46657, 47000), nil
}
