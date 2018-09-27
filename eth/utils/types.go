package utils

import (
	"github.com/loomnetwork/go-loom"
)

var (
	DeployEvm    = "deploy.evm"
	DeployPlugin = "deploy"
	CallEVM      = "call.evm"
	CallPlugin   = "call"
)

type EthBlockFilter struct {
	Addresses []loom.LocalAddress
	Topics    [][]string
}

type EthFilter struct {
	EthBlockFilter
	FromBlock string
	ToBlock   string
}
