package rpc

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func RunRPCProxyServer(listenPort, targetPort int32) error {
	proxyServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: rpcProxy(targetPort),
	}
	return proxyServer.ListenAndServe()
}

func rpcProxy(port int32) http.HandlerFunc {
	director := func(req *http.Request) {
		req.URL.Host = fmt.Sprintf("127.0.0.1:%d", port)
		req.URL.Scheme = "http"
		req.RequestURI = ""
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
