package rpc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tendermint/go-amino"
	rpccore "github.com/tendermint/tendermint/rpc/core"
	"github.com/tendermint/tendermint/rpc/lib/server"
)

func RPCServer(qsvc QueryService, logger log.TMLogger, bus *QueryEventBus, port int32) error {
	queryHandler := MakeQueryServiceHandler(qsvc, logger, bus)
	coreCodec := amino.NewCodec()

	wm := rpcserver.NewWebsocketManager(rpccore.Routes, coreCodec, rpcserver.EventSubscriber(bus))
	wm.SetLogger(logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)
	mux.Handle("/query", stripPrefix("/query", queryHandler)) //backwards compatibility
	mux.Handle("/queryws", queryHandler)
	rpcmux := http.NewServeMux()
	rpcserver.RegisterRPCFuncs(rpcmux, rpccore.Routes, coreCodec, logger)
	mux.Handle("/rpc", stripPrefix("/rpc", CORSMethodMiddleware(rpcmux)))

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())
	_, err := rpcserver.StartHTTPServer(
		fmt.Sprintf("tcp://0.0.0.0:%d", port), //todo get the address
		mux,
		logger,
		rpcserver.Config{MaxOpenConnections: 0},
	)
	return err
}

func stripPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			if p == "" {
				r2.URL.Path = "/"
			} else {
				r2.URL.Path = p
			}
			h.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
		}
	})
}

func CORSMethodMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if req.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		handler.ServeHTTP(w, req)
	})
}
