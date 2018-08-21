package throttle

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
)

func GetThrottleTxMiddleWare(maxAccessCount int64, sessionDuration int64, karmaEnabled bool, deployKarmaCount int64) loomchain.TxMiddlewareFunc {
	th := NewThrottle(maxAccessCount, sessionDuration, karmaEnabled, deployKarmaCount)
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
	) (res loomchain.TxHandlerResult, err error) {
		origin := auth.Origin(state.Context())
		if origin.IsEmpty() {
			return res, errors.New("transaction has no origin")
		}
		var tx loomchain.Transaction
		err1 := proto.Unmarshal(txBytes, &tx)
		limiterCtx, deployLimiterCtx, err, err1 := th.run(state, "ThrottleTxMiddleWare", tx.Id)

		if err != nil || err1 != nil {
			log.Error(err.Error())
			return res, err
		}

		if limiterCtx.Reached {
			message := fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime! Total access count %d", limiterCtx.Limit-limiterCtx.Remaining, limiterCtx.Limit, th.totalAccessCount[origin.String()])
			log.Error(message)
			return res, errors.New(message)
		}
		if tx.Id == 1 {
			if deployLimiterCtx.Reached {
				message := fmt.Sprintf("Out of deploy source count for current session: %d out of %d, Try after sometime! Total access count %d", deployLimiterCtx.Limit-deployLimiterCtx.Remaining, deployLimiterCtx.Limit, th.totaldeployKarmaCount[origin.String()])
				log.Error(message)
				return res, errors.New(message)
			}
		}

		return next(state, txBytes)
	})
}
