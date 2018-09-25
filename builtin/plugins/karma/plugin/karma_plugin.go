package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
)

var Contract = karma.Contract

func main() {
	plugin.Serve(Contract)
}
