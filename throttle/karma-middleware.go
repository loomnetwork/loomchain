package throttle

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/pkg/errors"
)

func GetKarmaMiddleWare(
	karmaEnabled bool,
	maxAccessCount int64,
	sessionDuration int64,
	registryVersion factory.RegistryVersion,
) loomchain.TxMiddlewareFunc {
	var createRegistry factory.RegistryFactoryFunc
	var registryObject registry.Registry
	th := NewThrottle(maxAccessCount, sessionDuration)
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
	) (res loomchain.TxHandlerResult, err error) {
		if !karmaEnabled {
			return next(state, txBytes)
		}

		origin := auth.Origin(state.Context())
		if origin.IsEmpty() {
			return res, errors.New("throttle: transaction has no origin")
		}

		var nonceTx lauth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx loomchain.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		if createRegistry == nil {
			createRegistry, err = factory.NewRegistryFactory(registryVersion)
			if err != nil {
				return res, errors.Wrap(err, "throttle: new registry factory")
			}
			registryObject = createRegistry(state)
		}

		if (0 == th.karmaContractAddress.Compare(loom.Address{})) {
			th.karmaContractAddress, err = registryObject.Resolve("karma")
			if err != nil {
				return next(state, txBytes)
			}
		}

		limiterCtx, deployLimiterCtx, err, err1 := th.run(state, "ThrottleTxMiddleWare", tx.Id, nonceTx.Sequence, false)

		if err != nil || err1 != nil {
			log.Error(err.Error())
			return res, err
		}

		if maxAccessCount > 0 {
			if limiterCtx.Reached {
				message := fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime! Total access count %d", limiterCtx.Limit-limiterCtx.Remaining, limiterCtx.Limit, th.totalAccessCount[origin.String()])
				log.Error(message)
				return res, errors.New(message)
			}
			if tx.Id == 1 {
				fmt.Println("Remaining", deployLimiterCtx.Remaining, "limit", deployLimiterCtx.Limit)
				message := fmt.Sprintf("Remaining %d limit %d", deployLimiterCtx.Remaining, deployLimiterCtx.Limit)
				log.Error(message)
				if deployLimiterCtx.Reached {
					//Not using limiting logic in this iteration
					message := fmt.Sprintf("Out of deploy source count for current session: %d out of %d, Try after sometime! Total access count %d", deployLimiterCtx.Limit-deployLimiterCtx.Remaining, deployLimiterCtx.Limit, th.totaldeployKarmaCount[origin.String()])
					log.Error(message)
					return res, errors.New(message)
				}
			}
		}

		return next(state, txBytes)
	})

}
