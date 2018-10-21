package rpc

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	rpccore "github.com/tendermint/tendermint/rpc/core"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
)

var cdc = amino.NewCodec()

//TODO I dislike how amino bleeds into places it shouldn't, lets see if we can push this back into tendermint
func init() {
	// RegisterAmino registers all crypto related types in the given (amino) codec.
	// These are all written here instead of
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		"tendermint/PubKeyEd25519", nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		"tendermint/PubKeySecp256k1", nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		"tendermint/PrivKeyEd25519", nil)
	cdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{},
		"tendermint/PrivKeySecp256k1", nil)

	cdc.RegisterInterface((*crypto.Signature)(nil), nil)
	cdc.RegisterConcrete(ed25519.SignatureEd25519{},
		"tendermint/SignatureEd25519", nil)
	cdc.RegisterConcrete(secp256k1.SignatureSecp256k1{},
		"tendermint/SignatureSecp256k1", nil)
}

func RPCServer(qsvc QueryService, logger log.TMLogger, bus *QueryEventBus, bindAddr string) error {
	queryHandler := MakeQueryServiceHandler(qsvc, logger, bus)

	wm := rpcserver.NewWebsocketManager(rpccore.Routes, cdc, rpcserver.EventSubscriber(bus))
	wm.SetLogger(logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)
	mux.Handle("/query/", stripPrefix("/query", queryHandler)) //backwards compatibility
	mux.Handle("/queryws", queryHandler)
	rpcmux := http.NewServeMux()
	rpcserver.RegisterRPCFuncs(rpcmux, rpccore.Routes, cdc, logger)
	mux.Handle("/rpc/", stripPrefix("/rpc", CORSMethodMiddleware(rpcmux)))
	mux.Handle("/rpc", stripPrefix("/rpc", CORSMethodMiddleware(rpcmux)))
	//mux.Handle("/rpc", CORSMethodMiddleware(rpcmux))

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())
	_, err := rpcserver.StartHTTPServer(
		bindAddr,
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
