package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
)

var Contract = chainconfig.Contract

func main() {
	plugin.Serve(Contract)
}
