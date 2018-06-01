package rpc

import (
	"net/http"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amino "github.com/tendermint/go-amino"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	"github.com/tendermint/tendermint/rpc/lib/types"
	"github.com/tendermint/tmlibs/pubsub"
	"golang.org/x/net/context"
)

// QueryService provides neccesary methods for the client to query appication states
type QueryService interface {
	Query(contract string, query []byte, vmType vm.VMType) ([]byte, error)
	Resolve(name string) (string, error)
	Nonce(key string) (uint64, error)
	Subscribe(wsCtx rpctypes.WSRPCContext, contract string) (*WSEmptyResult, error)
	UnSubscribe(wsCtx rpctypes.WSRPCContext) (*WSEmptyResult, error)
	TxReceipt(txHash []byte) ([]byte, error)
}
type queryEventBus struct {
	loomchain.SubscriptionSet
}

func (b *queryEventBus) Subscribe(ctx context.Context,
	subscriber string, query pubsub.Query, out chan<- interface{}) error {
	return nil
}

func (b *queryEventBus) Unsubscribe(ctx context.Context, subscriber string, query pubsub.Query) error {
	return nil
}

func (b *queryEventBus) UnsubscribeAll(ctx context.Context, subscriber string) error {
	log.Debug("Removing WS event subscriber", "address", subscriber)
	b.Remove(subscriber)
	return nil
}

// MakeQueryServiceHandler returns a http handler mapping to query service
func MakeQueryServiceHandler(svc QueryService, logger log.TMLogger) http.Handler {
	// set up websocket route
	codec := amino.NewCodec()
	wsmux := http.NewServeMux()
	routes := map[string]*rpcserver.RPCFunc{}
	routes["query"] = rpcserver.NewRPCFunc(svc.Query, "contract,query,vmType")
	routes["nonce"] = rpcserver.NewRPCFunc(svc.Nonce, "key")
	routes["subevents"] = rpcserver.NewWSRPCFunc(svc.Subscribe, "")
	routes["unsubevents"] = rpcserver.NewWSRPCFunc(svc.UnSubscribe, "")
	routes["resolve"] = rpcserver.NewRPCFunc(svc.Resolve, "name")
	routes["txreceipt"] = rpcserver.NewRPCFunc(svc.TxReceipt, "txHash")
	rpcserver.RegisterRPCFuncs(wsmux, routes, codec, logger)
	bus := &queryEventBus{}
	wm := rpcserver.NewWebsocketManager(routes, codec, rpcserver.EventSubscriber(bus))
	wsmux.HandleFunc("/queryws", wm.WebsocketHandler)

	// setup default route
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		wsmux.ServeHTTP(w, req)
	})

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
