package loom

import (
	"errors"
	"runtime/debug"

	"github.com/loomnetwork/loom/log"
)

type TxMiddleware interface {
	ProcessTx(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)
}

type TxMiddlewareFunc func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)

func (f TxMiddlewareFunc) ProcessTx(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
	return f(state, txBytes, next)
}

func MiddlewareTxHandler(
	middlewares []TxMiddleware,
	handler TxHandler,
) TxHandler {
	next := TxHandlerFunc(handler.ProcessTx)

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

var LogTxMiddleware = TxMiddlewareFunc(func(
	state State,
	txBytes []byte,
	next TxHandlerFunc,
) (TxHandlerResult, error) {
	// TODO: set some tx specific logging info
	return next(state, txBytes)
})
