package rpc

import (
	"log"
	"net/http"

	"github.com/loomnetwork/loom"
	llog "github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/plugin"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	tmcmn "github.com/tendermint/tmlibs/common"
)

// StateProvider interface is used by QueryServer to access the read-only application state
type StateProvider interface {
	ReadOnlyState() loom.State
}

// QueryServer provides the ability to query the current state of a contract via RPC
type QueryServer struct {
	StateProvider
	Host   string
	Logger llog.Logger
	Loader plugin.Loader
}

func (s *QueryServer) Start() error {
	smux := http.NewServeMux()
	routes := map[string]*rpcserver.RPCFunc{}
	routes["query"] = rpcserver.NewRPCFunc(s.queryRoute, "contract,query")
	rpcserver.RegisterRPCFuncs(smux, routes, s.Logger)
	wm := rpcserver.NewWebsocketManager(routes)
	smux.HandleFunc("/queryws", wm.WebsocketHandler)
	_, err := rpcserver.StartHTTPServer(s.Host, smux, s.Logger)
	if err != nil {
		return err
	}
	log.Printf("Query RPC Server running on %s", s.Host)
	return nil
}

func (s *QueryServer) RunForever() {
	tmcmn.TrapSignal(func() {
		// cleanup
	})
}

func (s *QueryServer) queryRoute(contract string, query []byte) ([]byte, error) {
	vm := &plugin.PluginVM{
		Loader: s.Loader,
		State:  s.StateProvider.ReadOnlyState(),
	}
	var caller loom.Address
	// TODO: marshal contract addr string -> loom.Address
	var contractAddr loom.Address
	return vm.StaticCall(caller, contractAddr, query)
}
