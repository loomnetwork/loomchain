package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dpos"
)

func main() {
	plugin.Serve(dpos.Contract)
}
