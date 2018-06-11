package main

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/cron"
)

var Contract = cron.Cron

func main() {
	plugin.Serve(Cron)
}
