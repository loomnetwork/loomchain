package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/pkg/errors"
)

func GetThrottleTxMiddleWare(
	deployEnabled func(blockHeight int64) bool,
	callEnabled func(blockHeight int64) bool,
	oracle loom.Address,
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		blockHeight := state.Block().Height
		isDeployEnabled := deployEnabled(blockHeight)
		isCallEnabled := callEnabled(blockHeight)

		if !isDeployEnabled || !isCallEnabled {
			origin := auth.Origin(state.Context())
			if origin.IsEmpty() {
				return res, errors.New("throttle: transaction has no origin")
			}

			var tx loomchain.Transaction
			if err := proto.Unmarshal(txBytes, &tx); err != nil {
				return res, errors.New("throttle: unmarshal tx")
			}

			if tx.Id == 1 && !deployEnabled(blockHeight) {
				if 0 != origin.Compare(oracle) {
					return res, errors.New("throttle: deploy transactions not enabled")
				}
			}

			if tx.Id == 2 && !callEnabled(blockHeight) {
				if 0 != origin.Compare(oracle) {
					return res, errors.New("throttle: call transactions not enabled")
				}
			}
		}
		return next(state, txBytes, isCheckTx)
	})
}
