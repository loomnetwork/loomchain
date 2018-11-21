package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/pkg/errors"
)

func GetKarmaMiddleWare(
	karmaEnabled bool,
	maxCallCount int64,
	sessionDuration int64,
	maxDeployCount int64,
	registryVersion factory.RegistryVersion,
) loomchain.TxMiddlewareFunc {
	var createRegistry factory.RegistryFactoryFunc
	var registryObject registry.Registry
	th := NewThrottle(sessionDuration, maxCallCount, maxDeployCount)
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

		// Oracle is not effected by karma restrictions
		karmaState, err := th.getKarmaState(state)
		if err != nil {
			return res, errors.Wrap(err, "getting karma state")
		}

		if karmaState.Has(karma.OracleKey) {
			var oraclePB types.Address
			if err := proto.Unmarshal(karmaState.Get(karma.OracleKey), &oraclePB); err != nil {
				return res, errors.Wrap(err, "unmarshal oracle")
			}
			if 0 == origin.Compare(loom.UnmarshalAddressPB(&oraclePB)) {
				return next(state, txBytes)
			}
		}

		totalKarma, err := th.getTotalKarma(state, origin, tx.Id)
		if err != nil {
			return res, errors.Wrap(err, "getting total karma")
		}

		if totalKarma == 0 {
			return res, errors.New("origin has no karma")
		}

		if tx.Id == 1 && maxDeployCount > 0 {
			err := th.runDeployThrottle(state, nonceTx.Sequence, origin)
			if err != nil {
				return res, errors.Wrap(err, "deploy karma throttle")
			}
		} else if tx.Id == 2 && maxCallCount > 0 {
			err := th.runCallThrottle(state, nonceTx.Sequence, totalKarma, origin)
			if err != nil {
				return res, errors.Wrap(err, "call karma throttle")
			}
		} else {
			return res, errors.Errorf("unknown tansaction id %d", tx.Id)
		}

		return next(state, txBytes)
	})

}
