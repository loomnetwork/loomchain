package throttle

import (
	"errors"
	"github.com/loomnetwork/loomchain"
	"log"
	"fmt"
	"github.com/loomnetwork/loomchain/auth"
)


var ThrottleTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (res loomchain.TxHandlerResult, err error)  {

	origin := auth.Origin(state.Context())
	if origin.IsEmpty() {
		return res, errors.New("transaction has no origin")
	}

	t := GetThrottle(origin)

	var accessCount int16 = 0

	if t.isSessionExpired() {
		t.setAccessCount(accessCount)
	} else {
		accessCount = t.incrementAccessCount()
	}

	log.Printf("Current session access count: %d out of %d\n", accessCount, t.maxAccessCount)

	message := fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!",  accessCount, t.maxAccessCount)

	if accessCount > 100 {
		panic(message)
	}


	return next(state, txBytes)
})