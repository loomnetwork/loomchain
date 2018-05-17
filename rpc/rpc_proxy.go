package rpc

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
)

func RunRPCProxyServer(listenPort, rpcPort int32, queryPort int32) error {
	proxyServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: rpcProxy(rpcPort, queryPort),
	}
	return proxyServer.ListenAndServe()
}

func rpcProxy(rpcPort int32, queryPort int32) http.HandlerFunc {
	director := func(req *http.Request) {
		if req.RequestURI == "/rpc" {
			req.URL.Host = fmt.Sprintf("127.0.0.1:%d", rpcPort)
			req.URL.Scheme = "http"
			req.RequestURI = ""
		} else if req.RequestURI == "/query" {
			req.URL.Host = fmt.Sprintf("127.0.0.1:%d", queryPort)
			req.URL.Scheme = "http"
			req.RequestURI = ""
		}
		parts := strings.SplitN(req.URL.Path, "/", 3)
		if len(parts) == 1 || len(parts) == 2 {
			req.URL.Path = "/"
		} else {
			req.URL.Path = "/" + parts[1]
		}
	}

	responseModifier := func(res *http.Response) error {
		res.Header.Add("Access-Control-Allow-Headers", "Content-Type")
		return nil
	}

	revProxy := httputil.ReverseProxy{Director: director, ModifyResponse: responseModifier}

	return func(w http.ResponseWriter, r *http.Request) {
		revProxy.ServeHTTP(w, r)
	}
}
