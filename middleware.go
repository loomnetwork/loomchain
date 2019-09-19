package loomchain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/state"
)

type TxHandler interface {
	ProcessTx(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error)
}

type TxHandlerFunc func(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error)

type TxHandlerResult struct {
	Data             []byte
	ValidatorUpdates []abci.Validator
	Info             string
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tendermint/libs/pubsub/query)
	Tags []common.KVPair
}

func (f TxHandlerFunc) ProcessTx(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	return f(s, txBytes, isCheckTx)
}

type TxMiddleware interface {
	ProcessTx(s state.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error)
}

type TxMiddlewareFunc func(s state.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error)

func (f TxMiddlewareFunc) ProcessTx(
	s state.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool,
) (TxHandlerResult, error) {
	return f(s, txBytes, next, isCheckTx)
}

type PostCommitHandler func(s state.State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error

type PostCommitMiddleware interface {
	ProcessTx(s state.State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool) error
}

type PostCommitMiddlewareFunc func(
	s state.State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool,
) error

func (f PostCommitMiddlewareFunc) ProcessTx(
	s state.State, txBytes []byte, res TxHandlerResult, next PostCommitHandler, isCheckTx bool,
) error {
	return f(s, txBytes, res, next, isCheckTx)
}

func MiddlewareTxHandler(
	middlewares []TxMiddleware,
	handler TxHandler,
	postMiddlewares []PostCommitMiddleware,
) TxHandler {
	postChain := func(s state.State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error { return nil }
	for i := len(postMiddlewares) - 1; i >= 0; i-- {
		m := postMiddlewares[i]
		localNext := postChain
		postChain = func(s state.State, txBytes []byte, res TxHandlerResult, isCheckTx bool) error {
			return m.ProcessTx(s, txBytes, res, localNext, isCheckTx)
		}
	}

	next := TxHandlerFunc(func(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
		result, err := handler.ProcessTx(s, txBytes, isCheckTx)
		if err != nil {
			return result, err
		}
		err = postChain(s, txBytes, result, isCheckTx)
		return result, err
	})

	for i := len(middlewares) - 1; i >= 0; i-- {
		m := middlewares[i]
		// Need local var otherwise infinite loop occurs
		nextLocal := next
		next = func(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
			return m.ProcessTx(s, txBytes, nextLocal, isCheckTx)
		}
	}

	return next
}

var NoopTxHandler = TxHandlerFunc(func(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
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
	s state.State,
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

	return next(s, txBytes, isCheckTx)
})

var LogTxMiddleware = TxMiddlewareFunc(func(
	s state.State,
	txBytes []byte,
	next TxHandlerFunc,
	isCheckTx bool,
) (TxHandlerResult, error) {
	// TODO: set some tx specific logging info
	return next(s, txBytes, isCheckTx)
})

var LogPostCommitMiddleware = PostCommitMiddlewareFunc(func(
	s state.State,
	txBytes []byte,
	res TxHandlerResult,
	next PostCommitHandler,
	isCheckTx bool,
) error {
	log.Default.Info("Tx processed", "result", res, "payload", base64.StdEncoding.EncodeToString(txBytes))
	return next(s, txBytes, res, isCheckTx)
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
	s state.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool,
) (r TxHandlerResult, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Tx", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	r, err = next(s, txBytes, isCheckTx)
	return
}
