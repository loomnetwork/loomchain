package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/blueprint"
)

var Contract = blueprint.Contract

func main() {
	plugin.Serve(Contract)
}
