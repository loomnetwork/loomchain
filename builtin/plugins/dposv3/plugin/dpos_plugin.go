package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
)

var Contract = dposv3.Contract

func main() {
	plugin.Serve(Contract)
}
