package loomchain

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/log"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type TxMiddleware interface {
	ProcessTx(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)
}

type TxMiddlewareFunc func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)

func (f TxMiddlewareFunc) ProcessTx(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
	return f(state, txBytes, next)
}

type PostCommitHandler func(state State, txBytes []byte, res TxHandlerResult) error

type PostCommitMiddleware interface {
	ProcessTx(state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler) error
}

type PostCommitMiddlewareFunc func(state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler) error

func (f PostCommitMiddlewareFunc) ProcessTx(state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler) error {
	return f(state, txBytes, res, next)
}

func MiddlewareTxHandler(
	middlewares []TxMiddleware,
	handler TxHandler,
	postMiddlewares []PostCommitMiddleware,
) TxHandler {
	postChain := func(state State, txBytes []byte, res TxHandlerResult) error { return nil }
	for i := len(postMiddlewares) - 1; i >= 0; i-- {
		m := postMiddlewares[i]
		localNext := postChain
		postChain = func(state State, txBytes []byte, res TxHandlerResult) error {
			return m.ProcessTx(state, txBytes, res, localNext)
		}
	}

	next := TxHandlerFunc(func(state State, txBytes []byte) (TxHandlerResult, error) {
		result, err := handler.ProcessTx(state, txBytes)
		if err != nil {
			return result, err
		}
		err = postChain(state, txBytes, result)
		return result, err
	})

	for i := len(middlewares) - 1; i >= 0; i-- {
		m := middlewares[i]
		// Need local var otherwise infinite loop occurs
		nextLocal := next
		next = func(state State, txBytes []byte) (TxHandlerResult, error) {
			return m.ProcessTx(state, txBytes, nextLocal)
		}
	}

	return next
}

var NoopTxHandler = TxHandlerFunc(func(state State, txBytes []byte) (TxHandlerResult, error) {
	return TxHandlerResult{}, nil
})

func rvalError(r interface{}) error {
	var err error
	switch x := r.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}
	return err
}

var RecoveryTxMiddleware = TxMiddlewareFunc(func(
	state State,
	txBytes []byte,
	next TxHandlerFunc,
) (res TxHandlerResult, err error) {
	defer func() {
		if rval := recover(); rval != nil {
			logger := log.Default
			logger.Error("Panic in TX Handler", "rvalue", rval)
			println(debug.Stack())
			err = rvalError(rval)
		}
	}()

	return next(state, txBytes)
})

var LogTxMiddleware = TxMiddlewareFunc(func(
	state State,
	txBytes []byte,
	next TxHandlerFunc,
) (TxHandlerResult, error) {
	// TODO: set some tx specific logging info
	return next(state, txBytes)
})

var LogPostCommitMiddleware = PostCommitMiddlewareFunc(func(
	state State,
	txBytes []byte,
	res TxHandlerResult,
	next PostCommitHandler,
) error {
	log.Default.Debug("Running post commit logger")
	log.Default.Info(string(txBytes))
	log.Default.Info(fmt.Sprintf("%+v", res))
	return next(state, txBytes, res)
})

// InstrumentingTxMiddleware maintains the state of metrics values internally
type InstrumentingTxMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

var _ TxMiddleware = &InstrumentingTxMiddleware{}

// NewInstrumentingTxMiddleware initializes the metrics and maintains the handler func
func NewInstrumentingTxMiddleware() TxMiddleware {
	// initialize metrcis
	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "tx_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "tx_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	return &InstrumentingTxMiddleware{
		requestCount:   requestCount,
		requestLatency: requestLatency,
	}
}

// ProcessTx capture metrics and implements TxMiddleware
func (m InstrumentingTxMiddleware) ProcessTx(state State, txBytes []byte, next TxHandlerFunc) (r TxHandlerResult, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Tx", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	r, err = next(state, txBytes)
	return
}

// InstrumentingEventHandler captures metrics and implements EventHandler
type InstrumentingEventHandler struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           EventHandler
}

var _ EventHandler = &InstrumentingEventHandler{}

// NewInstrumentingEventHandler initializes the metrics and maintains event handler
func NewInstrumentingEventHandler(next EventHandler) EventHandler {
	// initialize metrcis
	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "event_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "event_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	return &InstrumentingEventHandler{
		requestCount:   requestCount,
		requestLatency: requestLatency,
		next:           next,
	}
}

// Post captures the metrics
func (m InstrumentingEventHandler) Post(height uint64, e *EventData) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Post", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = m.next.Post(height, e)
	return
}

// EmitBlockTx captures the metrics
func (m InstrumentingEventHandler) EmitBlockTx(height uint64) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EmitBlockTx", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = m.next.EmitBlockTx(height)
	return
}

func (m InstrumentingEventHandler) SubscriptionSet() *SubscriptionSet {
	return nil
}

func (m InstrumentingEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return nil
}

func (m InstrumentingEventHandler) EthDepreciatedSubscriptionSet() *subs.EthDepreciatedSubscriptionSet {
	return nil
}
