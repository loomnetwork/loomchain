package rpc

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
)

// InstrumentingMiddleware wraps QuerySerice with metrics
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

// Query implements QueryService
func (m InstrumentingMiddleware) Query(contract string, query []byte) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "query", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Query(contract, query)
	return
}

// Nonce implements QueryService
func (m InstrumentingMiddleware) Nonce(key string) (resp uint64, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Nonce", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Nonce(key)
	return
}
