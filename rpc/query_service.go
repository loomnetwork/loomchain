package rpc

import (
	"net/http"

	"github.com/loomnetwork/loomchain/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amino "github.com/tendermint/go-amino"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	"github.com/loomnetwork/loomchain"
	"golang.org/x/net/context"
	"github.com/tendermint/tmlibs/pubsub"
)

// QueryService provides neccesary methods for the client to query appication states
type QueryService interface {
	Query(contract string, query []byte) ([]byte, error)
	Nonce(key string) (uint64, error)
}
type queryEventBus struct {
	loomchain.SubscriptionSet
}

func (b *queryEventBus) Subscribe(ctx context.Context,
	subscriber string, query pubsub.Query, out chan<- interface{}) error { return nil }

func (b *queryEventBus) Unsubscribe(ctx context.Context, subscriber string, query pubsub.Query) error { return nil }

func (b *queryEventBus) UnsubscribeAll(ctx context.Context, subscriber string) error {
	b.Remove(subscriber)
	return nil
}

// MakeQueryServiceHandler returns a http handler mapping to query service
func MakeQueryServiceHandler(svc QueryService, logger log.TMLogger) http.Handler {
	// set up websocket route
	codec := amino.NewCodec()
	wsmux := http.NewServeMux()
	routes := map[string]*rpcserver.RPCFunc{}
	routes["query"] = rpcserver.NewRPCFunc(svc.Query, "contract,query")
	routes["nonce"] = rpcserver.NewRPCFunc(svc.Nonce, "key")
	//	routes["events"] = rpcserver.NewRPCFunc(svc.Events, "key")
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
