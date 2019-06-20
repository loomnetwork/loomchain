package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

const (
	deployerStatePrefix = "ds"
)

type contractTxLimiter struct {
	// contract_address to limiting parametres structure
	contractToTierMap map[string]udw.Tier
	// track of no. of txns in previous blocks per contract ()
	contractToBlockTrx map[string]map[int64]int64
}

var TxLimiter *contractTxLimiter

func (txl *contractTxLimiter) isAccountLimitReached(account loom.Address) bool {
	//todo: check if limit is reached
	return true
}

// NewContractTxLimiterMiddleware add another tx limiter that limits how many CallTx(s) can be sent to an EVM contract within a pre-configured block range
func NewContractTxLimiterMiddleware(
	createUserDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {

		if !isCheckTx {
			return next(state, txBytes, isCheckTx)
		}

		var nonceTx auth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx loomchain.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		// Should not be initialized at every checkTx
		if TxLimiter == nil {
			ctx, err := createUserDeployerWhitelistCtx(state)
			if err != nil {
				return res, errors.Wrap(err, "throttle: context creation")
			}
			contractToTierMap, err := udw.GetContractTierMapping(ctx)
			if err != nil {
				return res, errors.Wrap(err, "throttle: contractToTierMap creation")
			}
			TxLimiter = &contractTxLimiter{contractToTierMap: contractToTierMap}
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
				if TxLimiter.isAccountLimitReached(loom.UnmarshalAddressPB(msg.To)) {
					return loomchain.TxHandlerResult{}, errors.New("tx limit reached, try again later")
				}
			}
		}

		return next(state, txBytes, isCheckTx)
	})
}
