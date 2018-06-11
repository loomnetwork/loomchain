package cron

import (
	"github.com/loomnetwork/loomchain"
)

func CronThrottleTxMiddleWare() loomchain.TxMiddlewareFunc {
	//TODO setup shared state

	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
	) (res loomchain.TxHandlerResult, err error) {

		//TODO check if its a cron tx, if so unwrap it, and pass it to another contract

		return next(state, txBytes)
	})
}
