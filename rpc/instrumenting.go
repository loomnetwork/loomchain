package rpc

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/tendermint/tendermint/rpc/lib/types"
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
func (m InstrumentingMiddleware) Query(caller, contract string, query []byte, vmType vm.VMType) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Query", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Query(caller, contract, query, vmType)
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

func (m InstrumentingMiddleware) Subscribe(wsCtx rpctypes.WSRPCContext, contracts []string) (*WSEmptyResult, error) {
	return m.next.Subscribe(wsCtx, contracts)
}

func (m InstrumentingMiddleware) UnSubscribe(wsCtx rpctypes.WSRPCContext, topic string) (*WSEmptyResult, error) {
	return m.next.UnSubscribe(wsCtx, topic)
}

func (m InstrumentingMiddleware) Resolve(name string) (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Resolve", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Resolve(name)
	return
}

func (m InstrumentingMiddleware) TxReceipt(txHash []byte) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "TxReceipt", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.TxReceipt(txHash)
	return
}

func (m InstrumentingMiddleware) GetCode(contract string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetCode", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetCode(contract)
	return
}

func (m InstrumentingMiddleware) GetLogs(filter string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetLogs", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetLogs(filter)
	return
}

func (m InstrumentingMiddleware) NewFilter(filter string) (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "NewFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.NewFilter(filter)
	return
}

func (m InstrumentingMiddleware) GetFilterChanges(id string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetFilterChanges", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetFilterChanges(id)
	return
}

func (m InstrumentingMiddleware) UninstallFilter(id string) (resp bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "UninstallFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.UninstallFilter(id)
	return
}
