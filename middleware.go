package loom

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
