package throttle

import (
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common"
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
	registryVersion factory.RegistryVersion,
) loomchain.TxMiddlewareFunc {
	var createRegistry factory.RegistryFactoryFunc
	var registryObject registry.Registry
	th := NewThrottle(sessionDuration, maxCallCount)
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		if !karmaEnabled {
			return next(state, txBytes, isCheckTx)
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
				return next(state, txBytes, isCheckTx)
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
				return next(state, txBytes, isCheckTx)
			}
		}

		originKarma, err := th.getKarmaForTransaction(state, origin, tx.Id)
		if err != nil {
			return res, errors.Wrap(err, "getting total karma")
		}

		if originKarma == nil || originKarma.Cmp(common.BigZero()) == 0 {
			return res, errors.New("origin has no karma of the appropiate type")
		}

		// Assume that if karma is more than maxint64
		// the user clearly has enough for a deploy or call tx,
		if 1 == originKarma.Cmp(loom.NewBigUIntFromInt(math.MaxInt64)) {
			return next(state, txBytes, isCheckTx)
		} else 	if !originKarma.IsInt64() {
			return res, errors.Wrapf(err, "cannot recognise karma total %v as an number", originKarma)
		}
		karmaTotal := originKarma.Int64()

		if tx.Id == deployId {
			var config ktypes.KarmaConfig
			if err := proto.Unmarshal(karmaState.Get(karma.OracleKey), &config); err != nil {
				return res, errors.Wrap(err, "unmarshal karma config")
			}
			if karmaTotal < config.MinKarmaToDeploy {
				return res, fmt.Errorf("not enough karma %v to depoy, required %v", karmaTotal, config.MinKarmaToDeploy)
			}
		} else if tx.Id == callId {
			if maxCallCount <= 0 {
				return res, errors.Errorf("max call count %d non positive", maxCallCount)
			}

			err := th.runThrottle(state, nonceTx.Sequence, origin, th.maxCallCount+karmaTotal, tx.Id, key)
			if err != nil {
				return res, errors.Wrap(err, "call karma throttle")
			}
		} else {
			return res, errors.Errorf("unknown transaction id %d", tx.Id)
		}
		return next(state, txBytes, isCheckTx)
	})

}
