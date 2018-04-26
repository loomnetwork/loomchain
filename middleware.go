package loom

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
