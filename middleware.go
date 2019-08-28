package loomchain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/loomchain/log"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type TxMiddleware interface {
	ProcessTx(state State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error)
}

type TxMiddlewareFunc func(state State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error)

func (f TxMiddlewareFunc) ProcessTx(
	state State, txBytes []byte, next TxHandlerFunc, isCheckTx bool,
) (TxHandlerResult, error) {
	return f(state, txBytes, next, isCheckTx)
}

type PostCommitHandler func(state State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error

type PostCommitMiddleware interface {
	ProcessTx(state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool) error
}

type PostCommitMiddlewareFunc func(
	state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool,
) error

func (f PostCommitMiddlewareFunc) ProcessTx(
	state State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool,
) error {
	return f(state, txBytes, res, next, isCheckTx)
}

func MiddlewareTxHandler(
	middlewares []TxMiddleware,
	handler TxHandler,
	postMiddlewares []PostCommitMiddleware,
) TxHandler {
	postChain := func(state State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error { return nil }
	for i := len(postMiddlewares) - 1; i >= 0; i-- {
		m := postMiddlewares[i]
		localNext := postChain
		postChain = func(state State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error {
			return m.ProcessTx(state, txBytes, res, localNext, isCheckTx)
		}
	}

	next := TxHandlerFunc(func(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
		result, err := handler.ProcessTx(state, txBytes, isCheckTx)
		if err != nil {
			return result, err
		}
		err = postChain(state, txBytes, result, isCheckTx)
		return result, err
	})

	for i := len(middlewares) - 1; i >= 0; i-- {
		m := middlewares[i]
		// Need local var otherwise infinite loop occurs
		nextLocal := next
		next = func(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
			return m.ProcessTx(state, txBytes, nextLocal, isCheckTx)
		}
	}

	return next
}

var NoopTxHandler = TxHandlerFunc(func(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
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
	isCheckTx bool,
) (res TxHandlerResult, err error) {
	defer func() {
		if rval := recover(); rval != nil {
			logger := log.Default
			logger.Error("Panic in TX Handler", "rvalue", rval)
			println(debug.Stack())
			err = rvalError(rval)
		}
	}()

	return next(state, txBytes, isCheckTx)
})

var LogTxMiddleware = TxMiddlewareFunc(func(
	state State,
	txBytes []byte,
	next TxHandlerFunc,
	isCheckTx bool,
) (TxHandlerResult, error) {
	// TODO: set some tx specific logging info
	return next(state, txBytes, isCheckTx)
})

var LogPostCommitMiddleware = PostCommitMiddlewareFunc(func(
	state State,
	txBytes []byte,
	res TxHandlerResult,
	next PostCommitHandler,
	isCheckTx bool,
) error {
	log.Default.Info("Tx processed", "result", res, "payload", base64.StdEncoding.EncodeToString(txBytes))
	return next(state, txBytes, res, isCheckTx)
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
		Namespace:  "loomchain",
		Subsystem:  "tx_service",
		Name:       "request_latency_microseconds",
		Help:       "Total duration of requests in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)

	return &InstrumentingTxMiddleware{
		requestCount:   requestCount,
		requestLatency: requestLatency,
	}
}

// ProcessTx capture metrics and implements TxMiddleware
func (m InstrumentingTxMiddleware) ProcessTx(
	state State, txBytes []byte, next TxHandlerFunc, isCheckTx bool,
) (r TxHandlerResult, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Tx", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	r, err = next(state, txBytes, isCheckTx)
	return
}
