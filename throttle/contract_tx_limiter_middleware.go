package throttle

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	ErrTxLimitReached = errors.New("tx limit reached, try again later")
)

var (
	tierMapLoadLatency                          metrics.Histogram
	contractTierMapLoadLatency                  metrics.Histogram
)

func init() {
	fieldKeys := []string{"method", "error"}
	tierMapLoadLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "contract_tx_limiter_middleware",
		Name:       "tier_map_load_latency_seconds",
		Help:       "Total time taken for Tier Map to Load in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
	contractTierMapLoadLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "contract_tx_limiter_middleware",
		Name:       "contract_tier_map_load_latency_seconds",
		Help:       "Total time taken for Contract Tier Map to Load in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
}

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
	contractStatsMap    map[string]*contractStats
	tierMap             map[udw.TierID]udw.Tier
	tierDataLastUpdated int64
}

type contractStats struct {
	txn         int64
	blockHeight int64
}

func (txl *contractTxLimiter) isAccountLimitReached(contractAddr loom.Address, curBlockHeight int64) bool {
	blockTx, ok := txl.contractStatsMap[contractAddr.String()]
	if !ok {
		return false
	}
	// if execution reaches here => tierID and tier are valid
	tierID := txl.contractToTierMap[contractAddr.String()]
	tier := txl.tierMap[tierID]
	if blockTx.blockHeight <= (curBlockHeight-int64(tier.BlockRange)) || int64(tier.MaxTxs) > blockTx.txn {
		return false
	}
	return true
}

func (txl *contractTxLimiter) updateState(contractAddr loom.Address, curBlockHeight int64) {
	blockTx, ok := txl.contractStatsMap[contractAddr.String()]
	tierID := txl.contractToTierMap[contractAddr.String()]
	tier := txl.tierMap[tierID]
	blockRange := int64(4096) // prevent divide by zero just in case tier doesn't have a range set
	if tier.BlockRange > 0 {
		blockRange = int64(tier.BlockRange)
	}
	if !ok || blockTx.blockHeight <= (curBlockHeight-blockRange) {
		// resetting the blockHeight to lower bound of range instead of curblockheight
		rangeStart := (((curBlockHeight - 1) / blockRange) * blockRange) + 1
		txl.contractStatsMap[contractAddr.String()] = &contractStats{1, rangeStart}
		return
	}
	blockTx.txn++
}

func LoadContractTierMap(ctx contractpb.StaticContext) (map[string]udwtypes.TierID, error) {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "LoadContractTierMap", "error", fmt.Sprint(err != nil)}
		contractTierMapLoadLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	contractToTierMap, err := udw.GetContractTierMapping(ctx)
	return contractToTierMap, err
}

func LoadTierMap(ctx contractpb.StaticContext) (map[udwtypes.TierID]udwtypes.Tier, error) {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "LoadTierMap", "error", fmt.Sprint(err != nil)}
		tierMapLoadLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	tierMap, err := udw.GetTierMap(ctx)
	return tierMap, err
}

// NewContractTxLimiterMiddleware creates a middleware function that limits how many call txs can be
// sent to an EVM contract within a pre-configured block range.
func NewContractTxLimiterMiddleware(cfg *ContractTxLimiterConfig,
	createUserDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	txl := &contractTxLimiter{
		contractStatsMap: make(map[string]*contractStats, 0),
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

		if msgTx.VmType != vm.VMType_EVM {
			return next(state, txBytes, isCheckTx)
		}

		if txl.contractToTierMap == nil ||
			(txl.contractDataLastUpdated+cfg.ContractDataRefreshInterval) < time.Now().Unix() {
			ctx, err := createUserDeployerWhitelistCtx(state)
			if err != nil {
				return res, errors.Wrap(err, "throttle: context creation")
			}
			contractToTierMap, err := LoadContractTierMap(ctx)
			if err != nil {
				return res, errors.Wrap(err, "throttle: contractToTierMap creation")
			}
			txl.contractToTierMap = contractToTierMap
			txl.contractDataLastUpdated = time.Now().Unix()
		}

		contractAddr := loom.UnmarshalAddressPB(msg.To)
		// contracts the limiter doesn't know about shouldn't be throttled
		contractTierID, ok := txl.contractToTierMap[contractAddr.String()]
		if !ok {
			return next(state, txBytes, isCheckTx)
		}
		if txl.tierMap == nil ||
			(txl.tierDataLastUpdated+cfg.TierDataRefreshInterval) < time.Now().Unix() {
			ctx, er := createUserDeployerWhitelistCtx(state)
			if er != nil {
				return res, errors.Wrap(err, "throttle: context creation")
			}
			txl.tierMap, err = LoadTierMap(ctx)
			if err != nil {
				return res, errors.Wrap(err, "throttle: GetTierMap error")
			}
			txl.tierDataLastUpdated = time.Now().Unix()
		}
		// ensure that tier corresponding to contract available in tierMap
		_, ok = txl.tierMap[contractTierID]
		if !ok {
			ctx, er := createUserDeployerWhitelistCtx(state)
			if er != nil {
				return res, errors.Wrap(err, "throttle: context creation")
			}
			tierInfo, er := udw.GetTierInfo(ctx, contractTierID)
			if er != nil {
				return res, errors.Wrap(err, "throttle: getTierInfo error")
			}
			txl.tierMap[contractTierID] = tierInfo
		}

		if txl.isAccountLimitReached(contractAddr, state.Block().Height) {
			return loomchain.TxHandlerResult{}, ErrTxLimitReached
		}
		txl.updateState(contractAddr, state.Block().Height)
		return next(state, txBytes, isCheckTx)
	})
}
