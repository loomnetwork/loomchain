package throttle

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"

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

		if tx.Id == callId {
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx", tx.Data)
			}
			var tx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &tx);  err != nil {
				return res, errors.Wrapf(err, "unmarshal call tx", msg.Data)
			}
			if tx.VmType == vm.VMType_EVM {
				if !karmaState.Has(karma.ContractActiveRecordKey(loom.UnmarshalAddressPB(msg.To))) {
					return res, fmt.Errorf("contract %s is not active evm", loom.UnmarshalAddressPB(msg.To).String())
				}
			}
		}

		if karmaState.Has(karma.OracleKey) {
			var oraclePB types.Address
			if err := proto.Unmarshal(karmaState.Get(karma.OracleKey), &oraclePB); err != nil {
				return res, errors.Wrap(err, "unmarshal oracle")
			}
			if 0 == origin.Compare(loom.UnmarshalAddressPB(&oraclePB)) {
				r, err := next(state, txBytes, isCheckTx)
				if !isCheckTx && err == nil && r.Info == utils.DeployEvm {
					dr := vm.DeployResponse{}
					if err := proto.Unmarshal(r.Data, &dr); err != nil {
						log.Warn("deploy repsonse does not unmarshal, %s", err.Error())
					}
					if err := karma.AddOwnedContract(karmaState, origin, loom.UnmarshalAddressPB(dr.Contract), state.Block().Height, nonceTx.Sequence); err != nil {
						log.Warn("adding contract to karma registry, %s", err.Error())
					}
				}
				return r, err
			}
		}

		originKarma, err := th.getTotalKarma(state, origin, tx.Id)
		if err != nil {
			return res, errors.Wrap(err, "getting total karma")
		}
		if originKarma == 0 {
			return res, errors.New("origin has no karma")
		}

		if tx.Id == deployId {
			err := th.runThrottle(state, nonceTx.Sequence, origin, originKarma, tx.Id, delpoyKey)
			if err != nil {
				return res, errors.Wrap(err, "deploy karma throttle")
			}
			r, err := next(state, txBytes, isCheckTx)
			if !isCheckTx && err == nil && r.Info == utils.DeployEvm {
				dr := vm.DeployResponse{}
				if err := proto.Unmarshal(r.Data, &dr); err != nil {
					return r, errors.Wrapf(err, "deploy response does not unmarshal, %v", dr)
				}
				if err := karma.AddOwnedContract(karmaState, origin, loom.UnmarshalAddressPB(dr.Contract), state.Block().Height, nonceTx.Sequence); err != nil {
					return r, errors.Wrapf(err,"adding contract to karma registry, %v", dr.Contract)
				}
			}
			return r, err

		} else if tx.Id == callId {
			if maxCallCount <= 0 {
				return res, errors.Errorf("max call count %d non positive", maxCallCount)
			}
			err := th.runThrottle(state, nonceTx.Sequence, origin, th.maxCallCount+originKarma, tx.Id, key)
			if err != nil {
				return res, errors.Wrap(err, "call karma throttle")
			}
		} else {
			return res, errors.Errorf("unknown transaction id %d", tx.Id)
		}

		return next(state, txBytes, isCheckTx)
	})

}
