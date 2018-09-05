package throttle

import (
	// "errors"
	"fmt"
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/loomchain/builtin/plugins/karma`
	`github.com/loomnetwork/loomchain/registry`
	`github.com/loomnetwork/loomchain/registry/factory`
	"github.com/pkg/errors"
	
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
)

func GetThrottleTxMiddleWare(maxAccessCount int64, sessionDuration int64, karmaEnabled bool, deployKarmaCount int64, registryVersion factory.RegistryVersion) loomchain.TxMiddlewareFunc {
	var createRegistry   factory.RegistryFactoryFunc
	var registryObject registry.Registry
	th := NewThrottle(maxAccessCount, sessionDuration, karmaEnabled, deployKarmaCount)
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
	) (res loomchain.TxHandlerResult, err error) {
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
		
		karmaState, err := th.getKarmaState(state)
		if err != nil {
			return res, errors.Wrap(err, "throttle: cannot find karma state")
		}

		
		origin := auth.Origin(state.Context())
		if origin.IsEmpty() {
			return res, errors.New("throttle: transaction has no origin")
		}
		var tx loomchain.Transaction
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}
		
		var karmaConfig karma.Config
		if karmaState.Has(karma.GetConfigKey()) {
			curConfigB := karmaState.Get(karma.GetConfigKey())
			err := proto.Unmarshal(curConfigB, &karmaConfig)
			if err != nil {
				return res, errors.Wrap(err, "throttle: getting karma config")
			}
		} else {
			return res, errors.New("throttle: karma config not found")
		}
		if !karmaConfig.Enabled {
			return next(state, txBytes)
		}
		
		if tx.Id == 1 && !karmaConfig.DeployEnabled {
			if  0 != origin.Compare(loom.Address{
				ChainID: karmaConfig.Oracle.ChainId,
				Local:   karmaConfig.Oracle.Local,
			}) {
				return res, errors.New("throttle: deploy  tx not enabled")
			}
		}
		
		if tx.Id == 2 && !karmaConfig.CallEnabled {
			if 0 != origin.Compare(loom.Address{
				ChainID: karmaConfig.Oracle.ChainId,
				Local:   karmaConfig.Oracle.Local,
			}) {
				return res, errors.New("throttle: call tx not enabled")
			}
		}
		
		limiterCtx, deployLimiterCtx, err, err1 := th.run(state, "ThrottleTxMiddleWare", tx.Id)

		if err != nil || err1 != nil {
			log.Error(err.Error())
			return res, err
		}
		
		if karmaConfig.SessionMaxAccessCount > 0 {
			if  limiterCtx.Reached {
				message := fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime! Total access count %d", limiterCtx.Limit-limiterCtx.Remaining, limiterCtx.Limit, th.totalAccessCount[origin.String()])
				log.Error(message)
				return res, errors.New(message)
			}
			if tx.Id == 1  {
				fmt.Println("Remaining",deployLimiterCtx.Remaining,"limit",deployLimiterCtx.Limit)
				message := fmt.Sprintf("Remaining %d limit %d", deployLimiterCtx.Remaining,"limit",deployLimiterCtx.Limit)
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