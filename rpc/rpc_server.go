package rpc

import (
	"net/http"
	"net/http/pprof"
	"net/url"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amino "github.com/tendermint/go-amino"
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
}

// RPCServer starts up HTTP servers that handle client requests.
func RPCServer(
	qsvc QueryService, chainID string, logger log.TMLogger, bus *QueryEventBus, bindAddr string,
	enableUnsafeRPC bool, unsafeRPCBindAddress string, pprofEnabled bool,
) error {
	queryHandler := MakeQueryServiceHandler(qsvc, logger, bus)
	hub := newHub()
	go hub.run()
	ethHandler := MakeEthQueryServiceHandler(logger, hub, createDefaultEthRoutes(qsvc, chainID))

	// Add the nonce route to the TM routes so clients can query the nonce from the /websocket
	// and /rpc endpoints.
	rpccore.Routes["nonce"] = rpcserver.NewRPCFunc(qsvc.Nonce, "key,account")

	wm := rpcserver.NewWebsocketManager(rpccore.Routes, cdc, rpcserver.EventSubscriber(bus))
	wm.SetLogger(logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)
	mux.Handle("/query/", stripPrefix("/query", queryHandler))
	mux.Handle("/query", stripPrefix("/query", queryHandler)) //backwards compatibility
	mux.Handle("/queryws", queryHandler)
	mux.Handle("/eth", ethHandler)
	rpcmux := http.NewServeMux()
	rpcserver.RegisterRPCFuncs(rpcmux, rpccore.Routes, cdc, logger)
	mux.Handle("/rpc/", stripPrefix("/rpc", CORSMethodMiddleware(rpcmux)))
	mux.Handle("/rpc", stripPrefix("/rpc", CORSMethodMiddleware(rpcmux)))

	listener, err := rpcserver.Listen(
		bindAddr,
		rpcserver.Config{MaxOpenConnections: 0},
	)
	if err != nil {
		return err
	}

	// setup debug route
	if pprofEnabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	}

	//TODO TM 0.26.0 has cors builtin, should we reuse it?
	/*
		var rootHandler http.Handler = mux
		if n.config.RPC.IsCorsEnabled() {
			corsMiddleware := cors.New(cors.Options{
				AllowedOrigins: n.config.RPC.CORSAllowedOrigins,
				AllowedMethods: n.config.RPC.CORSAllowedMethods,
				AllowedHeaders: n.config.RPC.CORSAllowedHeaders,
			})
			rootHandler = corsMiddleware.Handler(mux)
		}
	*/

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())

	go rpcserver.StartHTTPServer(
		listener,
		mux,
		logger,
	)

	if enableUnsafeRPC {
		unsafeLogger := logger.With("interface", "unsafe")
		unsafeHandler := MakeUnsafeQueryServiceHandler(unsafeLogger)
		unsafeListener, err := rpcserver.Listen(
			unsafeRPCBindAddress,
			rpcserver.Config{MaxOpenConnections: 0},
		)

		if err != nil {
			return errors.Wrap(err, "failed to start unsafe listener")
		}

		go rpcserver.StartHTTPServer(
			unsafeListener,
			CORSMethodMiddleware(unsafeHandler),
			unsafeLogger,
		)
	}

	return nil
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

		//		if req.Method == "OPTIONS" || req.Method == "GET" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		//		}

		handler.ServeHTTP(w, req)
	})
}
