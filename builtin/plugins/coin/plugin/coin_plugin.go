package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

func main() {
	plugin.Serve(coin.Contract)
}
