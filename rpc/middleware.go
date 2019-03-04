package rpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/loomnetwork/loomchain/log"
	"github.com/prometheus/client_golang/prometheus"
	types "github.com/tendermint/tendermint/rpc/lib/types"
)

var (
	requestDuration *prometheus.SummaryVec
)

func init() {

	requestDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "http_request_duration_seconds",
		Help: "Time (in seconds) spent serving HTTP requests.",
	}, []string{"method"},
	)

	prometheus.MustRegister(requestDuration)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var method string

		path := r.URL.Path
		stringparts := strings.Split(path, "/")
		_, ok := Routes.RouteMap[stringparts[len(stringparts)-1]]

		if !ok {
			// Read the Body content
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Error("Error reading request body")
			}
			// Restore the io.ReadCloser to its original state
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

			// JSONRPC request
			var req types.RPCRequest

			_ = json.Unmarshal(body, &req)

			if len(req.Method) > 0 {
				method = req.Method
			}
		} else {
			// HTTP endpoint
			method = stringparts[len(stringparts)-1]
		}

		if Routes.RouteMap[method] == true {
			start := time.Now()
			next.ServeHTTP(w, r)
			requestDuration.WithLabelValues(method).Observe(float64(time.Since(start).Seconds()))

		} else {
			next.ServeHTTP(w, r)

		}

	})
}
