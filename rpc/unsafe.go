package rpc

import (
	"net/http"

	"github.com/loomnetwork/loomchain/log"
)

func MakeUnsafeHandler(svc QueryService, logger log.TMLogger) http.Handler {

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("eth_accounts\n"))

		ac := svc.RawDump()

		w.Write([]byte(ac))
		return
	})
	return mux
}
