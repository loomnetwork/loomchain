package rpc

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/loomnetwork/loomchain/log"
	"github.com/tendermint/go-amino"
	rpccore "github.com/tendermint/tendermint/rpc/core"
	"github.com/tendermint/tendermint/rpc/lib/server"
	"net/http"
	"net/url"
	"strings"
)

func RPCServer(qsvc QueryService, logger log.TMLogger, bus *QueryEventBus, port int32) *http.Server {
	router := mux.NewRouter()
	router.Handle("/rpc", stripPrefix("/rpc", makeTendermintHandler(logger, bus)))
	router.Handle("/query", stripPrefix("/query", makeQueryServiceHandler(qsvc, logger, bus)))
	http.Handle("/", router)

	return &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%d", port /*46657 cfg.RPCPort*/),
	}
}

func makeTendermintHandler(logger log.TMLogger, bus *QueryEventBus) http.Handler {
	coreCodec := amino.NewCodec()
	muxt := http.NewServeMux()
	wm := rpcserver.NewWebsocketManager(rpccore.Routes, coreCodec, rpcserver.EventSubscriber(bus))
	wm.SetLogger(logger)
	muxt.HandleFunc("/websocket", wm.WebsocketHandler)
	rpcserver.RegisterRPCFuncs(muxt, rpccore.Routes, coreCodec, logger)
	return rpcserver.RecoverAndLogHandler(muxt, logger)
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
