package rpc

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

// ServerMetrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a RPC server.
type ServerMetrics struct {
	 requestDuration       *prom.HistogramVec
}

// NewServerMetrics returns a ServerMetrics object. Use a new instance of
// ServerMetrics when not using the default Prometheus metrics registry, for
// example when wanting to control which metrics are added to a registry as
// opposed to automatically adding metrics via init functions.
func NewServerMetrics() *ServerMetrics {

	return &ServerMetrics{
		// Prometheus histograms for requests.
		requestDuration : prom.NewHistogramVec(prom.HistogramOpts{
		Name:      "request_duration_seconds",
		Help:      "Time (in seconds) spent serving HTTP requests.",
		Buckets: prom.DefBuckets,

	}, []string{"method", "route", "status_code", "ws"}),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *ServerMetrics) Describe(ch chan<- *prom.Desc) {
	m.requestDuration.Describe(ch)
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *ServerMetrics) Collect(ch chan<- prom.Metric) {
	m.requestDuration.Collect(ch)
}