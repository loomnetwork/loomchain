package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	`github.com/loomnetwork/loomchain/builtin/plugins/config`
)

var Contract = config.Contract

func main() {
	plugin.Serve(Contract)
}
