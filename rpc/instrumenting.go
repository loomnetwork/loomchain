package rpc

import (
	"fmt"
	"github.com/go-kit/kit/metrics"
	"github.com/loomnetwork/loomchain/vm"
	"time"
)

// InstrumentingMiddleware implements QuerySerice interface
type InstrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           QueryService
}

// NewInstrumentingMiddleWare return a new pointer to the struct
func NewInstrumentingMiddleWare(reqCount metrics.Counter, reqLatency metrics.Histogram, next QueryService) *InstrumentingMiddleware {
	return &InstrumentingMiddleware{
		requestCount:   reqCount,
		requestLatency: reqLatency,
		next:           next,
	}
}

// Query calls service Query and captures metrics
func (m InstrumentingMiddleware) Query(contract string, query []byte, vmType vm.VMType) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Query", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Query(contract, query, vmType)
	return
}

// Nonce call service Nonce method and captures metrics
func (m InstrumentingMiddleware) Nonce(key string) (resp uint64, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Nonce", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Nonce(key)
	return
}
