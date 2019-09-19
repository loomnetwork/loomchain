package throttle

import (
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	lauth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

const karmaMiddlewareThrottleKey = "ThrottleTxMiddleWare"

func GetKarmaMiddleWare(
	karmaEnabled bool,
	maxCallCount int64,
	sessionDuration int64,
	createKarmaContractCtx func(s state.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	th := NewThrottle(sessionDuration, maxCallCount)
	return loomchain.TxMiddlewareFunc(func(
		s state.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		if !karmaEnabled {
			return next(s, txBytes, isCheckTx)
		}

		origin := auth.Origin(s.Context())
		if origin.IsEmpty() {
			return res, errors.New("throttle: transaction has no origin [get-karma]")
		}

		var nonceTx lauth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx loomchain.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		ctx, err := createKarmaContractCtx(s)
		if err != nil {
			return res, errors.Wrap(err, "failed to create Karma contract context")
		}

		if tx.Id == callId {
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
			}
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
		}

		// Oracle is not effected by karma restrictions
		oracleAddr, err := karma.GetOracleAddress(ctx)
		if err != nil {
			return res, errors.Wrap(err, "failed to obtain Karma Oracle address")
		}
		if oracleAddr != nil && origin.Compare(*oracleAddr) == 0 {
			r, err := next(s, txBytes, isCheckTx)
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

		originKarma, err := th.getKarmaForTransaction(ctx, origin, tx.Id)
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

		if tx.Id == deployId {
			config, err := karma.GetConfig(ctx)
			if err != nil {
				return res, errors.Wrap(err, "failed to load karma config")
			}
			if originKarmaTotal < config.MinKarmaToDeploy {
				return res, fmt.Errorf("not enough karma %v to depoy, required %v", originKarmaTotal, config.MinKarmaToDeploy)
			}
		} else if tx.Id == callId {
			if maxCallCount <= 0 {
				return res, errors.Errorf("max call count %d non positive", maxCallCount)
			}
			callCount := th.maxCallCount + originKarmaTotal
			if originKarmaTotal > math.MaxInt64-th.maxCallCount {
				callCount = math.MaxInt64
			}
			err := th.runThrottle(s, nonceTx.Sequence, origin, callCount, tx.Id, karmaMiddlewareThrottleKey)
			if err != nil {
				return res, errors.Wrap(err, "call karma throttle")
			}
		} else {
			return res, errors.Errorf("unknown transaction id %d", tx.Id)
		}

		r, err := next(s, txBytes, isCheckTx)
		if err != nil {
			return r, err
		}

		if tx.Id == deployId {
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
