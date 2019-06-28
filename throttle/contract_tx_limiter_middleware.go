package throttle

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

var (
	ErrTxLimitReached         = errors.New("tx limit reached, try again later")
	ErrContractNotWhitelisted = errors.New("contract not whitelisted")
)

type ContractTxLimiterConfig struct {
	// Enables the middleware
	Enabled bool
	// Number of seconds each refresh lasts
	RefreshInterval int64
}

func DefaultContractTxLimiterConfig() *ContractTxLimiterConfig {
	return &ContractTxLimiterConfig{
		Enabled:         false,
		RefreshInterval: 15 * 60,
	}
}

// Clone returns a deep clone of the config.
func (c *ContractTxLimiterConfig) Clone() *ContractTxLimiterConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type contractTxLimiter struct {
	// contract_address to limiting parametres structure
	contractToTierMap map[string]udw.TierID
	// track of no. of txns in previous blocks per contract
	contractToBlockTrx map[string]*blockTxn
	tierMap            map[udw.TierID]tierTrack
	lastUpdated        int64
}

type tierTrack struct {
	Tier        udw.Tier
	LastUpdated int64
}
type blockTxn struct {
	txn         int64
	blockHeight int64
}

var TxLimiter *contractTxLimiter

func (txl *contractTxLimiter) isAccountLimitReached(contractAddr loom.Address, curBlockHeight int64) bool {
	_, ok := txl.contractToBlockTrx[contractAddr.String()]
	if !ok {
		return false
	}
	tierID := txl.contractToTierMap[contractAddr.String()]
	tier := txl.tierMap[tierID].Tier
	blockTx := txl.contractToBlockTrx[contractAddr.String()]
	if int64(tier.MaxTx) > blockTx.txn {
		return false
	} else {
		return true
	}
}

func (txl *contractTxLimiter) updateState(contractAddr loom.Address, curBlockHeight int64) {
	blockTx, ok := txl.contractToBlockTrx[contractAddr.String()]
	if !ok {
		txl.contractToBlockTrx[contractAddr.String()] = &blockTxn{1, curBlockHeight}
		return
	}
	// reset blockTxn if block is out of range
	tierID := txl.contractToTierMap[contractAddr.String()]
	tier := txl.tierMap[tierID].Tier

	if blockTx.blockHeight < curBlockHeight-int64(tier.BlockRange) {
		blockTx.txn = 0
		blockTx.blockHeight = curBlockHeight
		return
	}
	blockTx.txn++
}

// NewContractTxLimiterMiddleware add another tx limiter that limits how many CallTx(s) can be sent to an EVM contract within a pre-configured block range
func NewContractTxLimiterMiddleware(cfg *ContractTxLimiterConfig,
	createUserDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	TxLimiter = &contractTxLimiter{
		contractToBlockTrx: make(map[string]*blockTxn, 0),
		tierMap:            make(map[udw.TierID]tierTrack, 0),
	}
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
		if tx.Id != callId {
			return next(state, txBytes, isCheckTx)
		}
		var msg vm.MessageTx
		if err := proto.Unmarshal(tx.Data, &msg); err != nil {
			return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
		}
		var msgTx vm.CallTx
		if err := proto.Unmarshal(msg.Data, &msgTx); err != nil {
			return res, errors.Wrapf(err, "unmarshal call tx %v", msg.Data)
		}
		if msgTx.VmType == vm.VMType_EVM {
			if TxLimiter.contractToTierMap == nil || TxLimiter.lastUpdated+cfg.RefreshInterval < time.Now().Unix() {
				ctx, err := createUserDeployerWhitelistCtx(state)
				if err != nil {
					return res, errors.Wrap(err, "throttle: context creation")
				}
				contractToTierMap, err := udw.GetContractTierMapping(ctx)
				if err != nil {
					return res, errors.Wrap(err, "throttle: contractToTierMap creation")
				}
				TxLimiter.contractToTierMap = contractToTierMap
				TxLimiter.lastUpdated = time.Now().Unix()
			}
			contractAddr := loom.UnmarshalAddressPB(msg.To)
			//check if contract in list
			tierID, ok := TxLimiter.contractToTierMap[contractAddr.String()]
			if !ok {
				return next(state, txBytes, isCheckTx)
			}

			tier := TxLimiter.tierMap[tierID].Tier

			// check if tierTrack have latest version if not update it
			tierTrackOb, ok := TxLimiter.tierMap[tier.TierID]
			if !ok || tierTrackOb.LastUpdated+cfg.RefreshInterval < time.Now().Unix() {
				ctx, er := createUserDeployerWhitelistCtx(state)
				if er != nil {
					return res, errors.Wrap(err, "throttle: context creation")
				}
				tierInfo, er := udw.GetTierInfo(ctx, tier.TierID)
				if er != nil {
					return res, errors.Wrap(err, "throttle: getTierInfo issue")
				}
				TxLimiter.tierMap[tier.TierID] = tierTrack{
					Tier:        tierInfo,
					LastUpdated: time.Now().Unix(),
				}
			}
			if TxLimiter.isAccountLimitReached(contractAddr, state.Block().Height) {
				return loomchain.TxHandlerResult{}, ErrTxLimitReached
			}
			TxLimiter.updateState(contractAddr, state.Block().Height)
		}
		return next(state, txBytes, isCheckTx)
	})
}
