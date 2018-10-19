package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
)

var Contract = dposv2.Contract

func main() {
	plugin.Serve(Contract)
}
