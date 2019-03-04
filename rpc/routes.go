package rpc


//http request latencies monitoring is enabled for calls which access tendermint
type RouteMonitor struct {
	RouteMap map[string]bool
}

var Routes RouteMonitor

//Routes which access tendermint blockstore
var RPCRoutes = []string{
	"evmtxreceipt",
	"getevmlogs",
	"getevmfilterchanges",
	"ethgetblockbynumber",
	"getevmblockbynumber",
	"ethgetblockbyhash",
	"getevmblockbyhash",
	"ethgettransactionreceipt",
	"ethgetblocktransactioncountbyhash",
	"ethgetblocktransactioncountbynumber",
	"ethgettransactionbyblockhashandindex",
	"ethgettransactionbyblocknumberandindex",
	"ethgetlogs",
}

func init() {

	Routes.RouteMap = make(map[string]bool)
	for _,route := range RPCRoutes {
		Routes.RouteMap[route] = true
	}
}



