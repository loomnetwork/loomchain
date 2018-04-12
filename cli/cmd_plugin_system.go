package cli

import "github.com/loomnetwork/loom/client"

// CmdPluginSystem interface is used by command plugins to hook into the Loom admin CLI.
type CmdPluginSystem interface {
	// GetClient returns a DAppChainClient that can be used to commit txs to a Loom DAppChain.
	GetClient(nodeURI string) (client.DAppChainClient, error)
}

type cmdPluginSystem struct {
}

func NewCmdPluginSystem() CmdPluginSystem {
	return &cmdPluginSystem{}
}

func (ps *cmdPluginSystem) GetClient(nodeURI string) (client.DAppChainClient, error) {
	// TODO: cache the client instead of creating a new instance every time
	return client.NewDAppChainRPCClient(nodeURI), nil
}
