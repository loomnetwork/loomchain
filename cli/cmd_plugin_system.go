package cli

import (
	lp "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain/client"
)

// Implements the CmdPluginSystem interface used by cmd plugins
type cmdPluginSystem struct {
}

func NewCmdPluginSystem() lp.CmdPluginSystem {
	return &cmdPluginSystem{}
}

func (ps *cmdPluginSystem) GetClient(host string, rpcPort int, queryPort int) (lp.DAppChainClient, error) {
	// TODO: cache the client instead of creating a new instance every time
	return client.NewDAppChainRPCClient(host, int32(rpcPort), int32(queryPort)), nil
}
