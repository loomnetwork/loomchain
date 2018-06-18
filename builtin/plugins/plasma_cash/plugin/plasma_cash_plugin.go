package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash"
)

var Contract = plasma_cash.Contract

func main() {
	plugin.Serve(Contract)
}
