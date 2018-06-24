package rpc

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	"github.com/tendermint/tendermint/rpc/lib/types"
	"golang.org/x/net/context"
)

// QueryService provides neccesary methods for the client to query appication states
type QueryService interface {
	Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error)
	Resolve(name string) (string, error)
	Nonce(key string) (uint64, error)
	Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error)
	UnSubscribe(wsCtx rpctypes.WSRPCContext, topics string) (*WSEmptyResult, error)
	TxReceipt(txHash []byte) ([]byte, error)
	GetCode(contract string) ([]byte, error)
	GetLogs(filter string) ([]byte, error)
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
	b.Purge(subscriber)
	return nil
}

func QueryServiceWSManager(routes map[string]*rpcserver.RPCFunc) *rpcserver.WebsocketManager {
	codec := amino.NewCodec()
	bus := &queryEventBus{}
	return rpcserver.NewWebsocketManager(routes, codec, rpcserver.EventSubscriber(bus))
}

func QueryServiceRPCRoutes(svc QueryService) map[string]*rpcserver.RPCFunc {
	routes := map[string]*rpcserver.RPCFunc{}
	routes["query"] = rpcserver.NewRPCFunc(svc.Query, "caller,contract,query,vmType")
	routes["nonce"] = rpcserver.NewRPCFunc(svc.Nonce, "key")
	routes["subevents"] = rpcserver.NewWSRPCFunc(svc.Subscribe, "topics")
	routes["unsubevents"] = rpcserver.NewWSRPCFunc(svc.UnSubscribe, "topic")
	routes["resolve"] = rpcserver.NewRPCFunc(svc.Resolve, "name")
	routes["txreceipt"] = rpcserver.NewRPCFunc(svc.TxReceipt, "txHash")
	routes["getcode"] = rpcserver.NewRPCFunc(svc.GetCode, "contract")
	routes["getlogs"] = rpcserver.NewRPCFunc(svc.GetLogs, "filter")
	return routes
}

// MakeQueryServiceHandler returns a http handler mapping to query service
func MakeQueryServiceHandler(svc QueryService, logger log.TMLogger) http.Handler {
	// set up websocket route
	codec := amino.NewCodec()
	wsmux := http.NewServeMux()
	routes := QueryServiceRPCRoutes(svc)
	rpcserver.RegisterRPCFuncs(wsmux, routes, codec, logger)
	wm := QueryServiceWSManager(routes)
	wsmux.HandleFunc("/ws", wm.WebsocketHandler)
	// setup default route
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		debugReq, _ := httputil.DumpRequest(req, true)
		log.Debug("query handler", "request", string(debugReq))
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		// from https://go-review.googlesource.com/c/go/+/36483
		r2 := new(http.Request)
		*r2 = *req
		r2.URL = new(url.URL)
		*r2.URL = *req.URL
		parts := rmEmpty(strings.SplitN(req.URL.Path, "/", 3))
		if len(parts) > 1 {
			r2.URL.Path = "/" + parts[1]
		} else {
			r2.URL.Path = "/"
		}
		wsmux.ServeHTTP(w, r2)
	})

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
