package loom

import (
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
)

// NodeProxy proxies a subset of the Tendermint REST/RPC/WS API to a DAppChain node.
type NodeProxy struct {
	rpcClient rpcclient.Client
}

func NewNodeProxy(c rpcclient.Client) *NodeProxy {
	return &NodeProxy{
		rpcClient: c,
	}
}

// RPCRoutes returns a subset of RPC handlers for the Tendermint REST/RPC/WS API.
func (np *NodeProxy) RPCRoutes() map[string]*rpcserver.RPCFunc {
	return map[string]*rpcserver.RPCFunc{
		"broadcast_tx_commit": rpcserver.NewRPCFunc(np.rpcClient.BroadcastTxCommit, "tx"),
	}
}
