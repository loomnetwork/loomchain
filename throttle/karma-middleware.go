package throttle

import (
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/auth/keys"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/eth/utils"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/txhandler"
	"github.com/loomnetwork/loomchain/vm"
)

const karmaMiddlewareThrottleKey = "ThrottleTxMiddleWare"

func GetKarmaMiddleWare(
	karmaEnabled bool,
	maxCallCount int64,
	sessionDuration int64,
	createKarmaContractCtx func(state appstate.State) (contractpb.Context, error),
) txhandler.TxMiddlewareFunc {
	th := NewThrottle(sessionDuration, maxCallCount)
	return txhandler.TxMiddlewareFunc(func(
		state appstate.State,
		txBytes []byte,
		next txhandler.TxHandlerFunc,
		isCheckTx bool,
	) (res txhandler.TxHandlerResult, err error) {
		if !karmaEnabled {
			return next(state, txBytes, isCheckTx)
		}

		origin := keys.Origin(state.Context())
		if origin.IsEmpty() {
			return res, errors.New("throttle: transaction has no origin [get-karma]")
		}

		var nonceTx lauth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx types.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		var msg vm.MessageTx
		if err := proto.Unmarshal(tx.Data, &msg); err != nil {
			return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
		}

		ctx, err := createKarmaContractCtx(state)
		if err != nil {
			return res, errors.Wrap(err, "failed to create Karma contract context")
		}

		var isDeployTx bool
		switch types.TxID(tx.Id) {
		case types.TxID_CALL:
			isDeployTx = false
			var tx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &tx); err != nil {
				return res, errors.Wrapf(err, "unmarshal call tx %v", msg.Data)
			}
			if tx.VmType == vm.VMType_EVM {
				isActive, err := karma.IsContractActive(ctx, loom.UnmarshalAddressPB(msg.To))
				if err != nil {
					return res, errors.Wrapf(err, "determining if contract %v is active", loom.UnmarshalAddressPB(msg.To).String())
				}
				if !isActive {
					return res, fmt.Errorf("contract %s is not active", loom.UnmarshalAddressPB(msg.To).String())
				}
			}

		case types.TxID_DEPLOY:
			isDeployTx = true

		case types.TxID_ETHEREUM:
			isDeployTx, err = isEthDeploy(msg.Data)
			if err != nil {
				return res, err
			}
			if !isDeployTx {
				isActive, err := karma.IsContractActive(ctx, loom.UnmarshalAddressPB(msg.To))
				if err != nil {
					return res, errors.Wrapf(err, "determining if contract %v is active", loom.UnmarshalAddressPB(msg.To).String())
				}
				if !isActive {
					return res, fmt.Errorf("contract %s is not active", loom.UnmarshalAddressPB(msg.To).String())
				}
			}

		default:
			return next(state, txBytes, isCheckTx)
		}

		// Oracle is not effected by karma restrictions
		oracleAddr, err := karma.GetOracleAddress(ctx)
		if err != nil {
			return res, errors.Wrap(err, "failed to obtain Karma Oracle address")
		}
		if oracleAddr != nil && origin.Compare(*oracleAddr) == 0 {
			r, err := next(state, txBytes, isCheckTx)
			if err != nil {
				return r, err
			}
			if !isCheckTx && r.Info == utils.DeployEvm {
				dr := vm.DeployResponse{}
				if err := proto.Unmarshal(r.Data, &dr); err != nil {
					return r, errors.Wrapf(err, "deploy response %s does not unmarshal", string(r.Data))
				}
				if err := karma.AddOwnedContract(ctx, origin, loom.UnmarshalAddressPB(dr.Contract)); err != nil {
					return r, errors.Wrapf(err, "adding contract %s to karma registry", dr.Contract.String())
				}
			}
			return r, nil
		}

		originKarma, err := th.getKarmaForTransaction(ctx, origin, isDeployTx)
		if err != nil {
			return res, errors.Wrap(err, "getting total karma")
		}

		if originKarma == nil || originKarma.Cmp(common.BigZero()) == 0 {
			return res, errors.New("origin has no karma of the appropriate type")
		}

		var originKarmaTotal int64
		// If karma is more than maxint64, treat as maxint64 as both should be enough
		if 1 == originKarma.Cmp(loom.NewBigUIntFromInt(math.MaxInt64)) {
			originKarmaTotal = math.MaxInt64
		} else if !originKarma.IsInt64() {
			return res, errors.Wrapf(err, "cannot recognise karma total %v as an number", originKarma)
		} else {
			originKarmaTotal = originKarma.Int64()
		}

		if isDeployTx {
			config, err := karma.GetConfig(ctx)
			if err != nil {
				return res, errors.Wrap(err, "failed to load karma config")
			}
			if originKarmaTotal < config.MinKarmaToDeploy {
				return res, fmt.Errorf("not enough karma %v to depoy, required %v", originKarmaTotal, config.MinKarmaToDeploy)
			}
		} else {
			if maxCallCount <= 0 {
				return res, errors.Errorf("max call count %d non positive", maxCallCount)
			}
			callCount := th.maxCallCount + originKarmaTotal
			if originKarmaTotal > math.MaxInt64-th.maxCallCount {
				callCount = math.MaxInt64
			}
			err := th.runThrottle(state, nonceTx.Sequence, origin, callCount, tx.Id, karmaMiddlewareThrottleKey)
			if err != nil {
				return res, errors.Wrap(err, "call karma throttle")
			}
		}

		r, err := next(state, txBytes, isCheckTx)
		if err != nil {
			return r, err
		}

		if isDeployTx {
			if !isCheckTx && r.Info == utils.DeployEvm {
				dr := vm.DeployResponse{}
				if err := proto.Unmarshal(r.Data, &dr); err != nil {
					return r, errors.Wrapf(err, "deploy response does not unmarshal, %v", dr)
				}
				if err := karma.AddOwnedContract(ctx, origin, loom.UnmarshalAddressPB(dr.Contract)); err != nil {
					return r, errors.Wrapf(err, "adding contract to karma registry, %v", dr.Contract)
				}
			}
		}
		return r, nil
	})
}
