package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
)

var Contract = address_mapper.Contract

func main() {
	plugin.Serve(Contract)
}
