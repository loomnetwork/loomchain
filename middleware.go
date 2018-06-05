package loomchain

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/loomchain/log"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"bytes"
	"encoding/binary"
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
			logger := log.Root
			logger.Error("Panic in TX Handler", "rvalue", rval)
			println(debug.Stack())
			err = rvalError(rval)
		}
	}()

	return next(state, txBytes)
})

func startSessionTime(state State) (int64) {
	fmt.Println("----- No session found -----")

	sessionTime := time.Now().Unix()

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, sessionTime)
	if err != nil {
		panic(err)
	}

	state.Set([]byte("session-start-time"), buf.Bytes())

	return int64(binary.BigEndian.Uint64(state.Get([]byte("session-start-time"))))
}

func getSessionTime(state State) (int64) {
	return int64(binary.BigEndian.Uint64(state.Get([]byte("session-start-time"))))
}

func isSessionExpired(sessionStartTime, currentTime int64) (bool) {
	// TODO: current session time limit 10 minutes
	var sessionSize int64 = 600
	return sessionStartTime + sessionSize <= currentTime
}

func setSessionAccessCount(state State, accessCount int16) {

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, accessCount)
	if err != nil {
		panic(err)
	}

	state.Set([]byte("session-access-count"), buf.Bytes())
}

func getSessionAccessCount(state State) (int16) {
	return int16(binary.BigEndian.Uint16(state.Get([]byte("session-access-count"))))
}

var ThrottleTxMiddleware = TxMiddlewareFunc(func(
	state State,
	txBytes []byte,
	next TxHandlerFunc,
) (TxHandlerResult, error) {
	fmt.Println("------------------------------------------------------------")
	fmt.Println("ThrottleTxMiddleware")

	currentTime := time.Now().Unix()

	var accessCount int16
	var sessionStartTime int64
	if state.Has([]byte("session-start-time")) {
		sessionStartTime = getSessionTime(state)
	}else{
		sessionStartTime = startSessionTime(state)
		setSessionAccessCount(state, 0)
	}
	fmt.Println("start time: ",sessionStartTime)

	if isSessionExpired(sessionStartTime, currentTime) {
		fmt.Println("session expired:")
		setSessionAccessCount(state, 0)
	} else {
		accessCount = getSessionAccessCount(state) + 1
		setSessionAccessCount(state, accessCount)
	}

	fmt.Println("---------------------- Current access count: ", accessCount)

	if accessCount > 100 {
		fmt.Println("---------------------- Ran out of access count: ", accessCount)
		fmt.Println(accessCount)
	}

	fmt.Println("------------------------------------------------------------")


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
	log.Root.Debug("Running post commit logger")
	log.Root.Info(string(txBytes))
	log.Root.Info(fmt.Sprintf("%+v", res))
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
func (m InstrumentingEventHandler) Post(state State, e *EventData) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Post", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = m.next.Post(state, e)
	return
}

// EmitBlockTx captures the metrics
func (m InstrumentingEventHandler) EmitBlockTx(height int64) (err error) {
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
