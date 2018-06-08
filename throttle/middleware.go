package throttle

import (
	"errors"
	"github.com/loomnetwork/loomchain"
	"log"
	"fmt"
	"github.com/loomnetwork/loomchain/auth"
)


func GetThrottleTxMiddleWare(th *Throttle) (loomchain.TxMiddlewareFunc) {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
	) (res loomchain.TxHandlerResult, err error)  {

		origin := auth.Origin(state.Context())
		if origin.IsEmpty() {
			return res, errors.New("transaction has no origin")
		}

		th.setOriginContext(origin)

		var accessCount int16 = 0

		if th.isSessionExpired() {
			th.setAccessCount(accessCount)
		} else {
			accessCount = th.incrementAccessCount()
		}

		log.Printf("Current session access count: %d out of %d\n", accessCount, th.maxAccessCount)

		message := fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!",  accessCount, th.maxAccessCount)

		if accessCount > th.maxAccessCount {
			errors.New(message)
		}


		return next(state, txBytes)
	})
}