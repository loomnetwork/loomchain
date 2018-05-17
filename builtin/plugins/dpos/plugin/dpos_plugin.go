package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
)

var Contract = dpos.Contract

func main() {
	plugin.Serve(Contract)
}
