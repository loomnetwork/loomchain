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
	ContractDataRefreshInterval int64
	TierDataRefreshInterval     int64
}

func DefaultContractTxLimiterConfig() *ContractTxLimiterConfig {
	return &ContractTxLimiterConfig{
		Enabled:                     false,
		ContractDataRefreshInterval: 15 * 60,
		TierDataRefreshInterval:     15 * 60,
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
	contractToTierMap       map[string]udw.TierID
	contractDataLastUpdated int64
	// track of no. of txns in previous blocks per contract
	contractToBlockTrx  map[string]*blockTxn
	tierMap             map[udw.TierID]udw.Tier
	tierDataLastUpdated int64
}

type blockTxn struct {
	txn         int64
	blockHeight int64
}

var TxLimiter *contractTxLimiter

func (txl *contractTxLimiter) isAccountLimitReached(contractAddr loom.Address, curBlockHeight int64) bool {
	blockTx, ok := txl.contractToBlockTrx[contractAddr.String()]
	if !ok {
		return false
	}
	// if execution reaches here => tierID and tier are valid
	tierID := txl.contractToTierMap[contractAddr.String()]
	tier := txl.tierMap[tierID]
	// reset contractToBlockTrx if curBlock is out of range
	if blockTx.blockHeight <= curBlockHeight-int64(tier.BlockRange) {
		blockHeight := (curBlockHeight / int64(tier.BlockRange)) * int64(tier.BlockRange)
		blockTx = &blockTxn{0, blockHeight}
		txl.contractToBlockTrx[contractAddr.String()] = blockTx
	}
	if int64(tier.MaxTx) > blockTx.txn {
		return false
	} else {
		return true
	}
}

func (txl *contractTxLimiter) updateState(contractAddr loom.Address, curBlockHeight int64) {
	blockTx, ok := txl.contractToBlockTrx[contractAddr.String()]
	if !ok {
		tierID := txl.contractToTierMap[contractAddr.String()]
		tier := txl.tierMap[tierID]
		// resetting the blockHeight to lower bound of range instead of curblockheight
		blockHeight := (curBlockHeight / int64(tier.BlockRange)) * int64(tier.BlockRange)
		txl.contractToBlockTrx[contractAddr.String()] = &blockTxn{1, blockHeight}
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
			if TxLimiter.contractToTierMap == nil || TxLimiter.contractDataLastUpdated+cfg.ContractDataRefreshInterval < time.Now().Unix() {
				ctx, err := createUserDeployerWhitelistCtx(state)
				if err != nil {
					return res, errors.Wrap(err, "throttle: context creation")
				}
				contractToTierMap, err := udw.GetContractTierMapping(ctx)
				if err != nil {
					return res, errors.Wrap(err, "throttle: contractToTierMap creation")
				}
				TxLimiter.contractToTierMap = contractToTierMap
				TxLimiter.contractDataLastUpdated = time.Now().Unix()
			}
			contractAddr := loom.UnmarshalAddressPB(msg.To)
			//check if contract in list
			contractTierID, ok := TxLimiter.contractToTierMap[contractAddr.String()]
			if !ok {
				return next(state, txBytes, isCheckTx)
			}
			if TxLimiter.tierMap == nil || TxLimiter.tierDataLastUpdated+cfg.TierDataRefreshInterval <
				time.Now().Unix() {
				ctx, er := createUserDeployerWhitelistCtx(state)
				if er != nil {
					return res, errors.Wrap(err, "throttle: context creation")
				}
				TxLimiter.tierMap, err = udw.GetTierMap(ctx)
				if err != nil {
					return res, errors.Wrap(err, "throttle: GetTierMap error")
				}
				TxLimiter.tierDataLastUpdated = time.Now().Unix()
			}

			// check if tier corresponding to contract available in tierMap
			_, ok = TxLimiter.tierMap[contractTierID]
			if !ok {
				ctx, er := createUserDeployerWhitelistCtx(state)
				if er != nil {
					return res, errors.Wrap(err, "throttle: context creation")
				}
				tierInfo, er := udw.GetTierInfo(ctx, contractTierID)
				if er != nil {
					return res, errors.Wrap(err, "throttle: getTierInfo error")
				}
				TxLimiter.tierMap[contractTierID] = tierInfo
			}
			if TxLimiter.isAccountLimitReached(contractAddr, state.Block().Height) {
				return loomchain.TxHandlerResult{}, ErrTxLimitReached
			}
			TxLimiter.updateState(contractAddr, state.Block().Height)
		}
		return next(state, txBytes, isCheckTx)
	})
}
