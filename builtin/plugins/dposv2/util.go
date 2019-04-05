package dposv2

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var TierMap = map[uint64]LocktimeTier{
	0: TIER_ZERO,
	1: TIER_ONE,
	2: TIER_TWO,
	3: TIER_THREE,
}

var TierLocktimeMap = map[LocktimeTier]uint64{
	TIER_ZERO:  1209600,  // two weeks
	TIER_ONE:   7884000,  // three months
	TIER_TWO:   15768000, // six months
	TIER_THREE: 31536000, // one year
}

var TierBonusMap = map[LocktimeTier]loom.BigUInt{
	TIER_ZERO:  loom.BigUInt{big.NewInt(10000)}, // two weeks
	TIER_ONE:   loom.BigUInt{big.NewInt(15000)}, // three months
	TIER_TWO:   loom.BigUInt{big.NewInt(20000)}, // six months
	TIER_THREE: loom.BigUInt{big.NewInt(40000)}, // one year
}

// frac is expressed in basis points or millionths of basis points
func CalculateFraction(frac loom.BigUInt, total loom.BigUInt, granular bool) loom.BigUInt {
	return CalculatePreciseFraction(frac, total, granular)
}

// frac is expressed in billionths
func CalculatePreciseFraction(frac loom.BigUInt, total loom.BigUInt, granular bool) loom.BigUInt {
	denom := basisPoints
	if granular {
		frac = basisPointsToBillionths(frac)
		denom = billionth
	}
	updatedAmount := loom.BigUInt{big.NewInt(0)}
	updatedAmount.Mul(&total, &frac)
	updatedAmount.Div(&updatedAmount, &denom)
	return updatedAmount
}

func calculateShare(delegation loom.BigUInt, total loom.BigUInt, rewards loom.BigUInt, granular bool) loom.BigUInt {
	frac := common.BigZero()
	denom := &basisPoints
	if granular {
		denom = &billionth
	}
	if !common.IsZero(total) {
		frac.Mul(&delegation, denom)
		frac.Div(frac, &total)
	}
	return CalculatePreciseFraction(*frac, rewards, granular)
}

func scientificNotation(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

func calculateTierLocktime(tier LocktimeTier, electionCycleLength uint64) uint64 {
	if tier == TIER_ZERO && electionCycleLength < TierLocktimeMap[tier] {
		return electionCycleLength
	}
	return TierLocktimeMap[tier]
}

func calculateWeightedDelegationAmount(delegation Delegation, v2_1 bool) loom.BigUInt {
	bonusPercentage := TierBonusMap[delegation.LocktimeTier]

	return CalculateFraction(bonusPercentage, delegation.Amount.Value, v2_1)
}

func basisPointsToBillionths(bps loom.BigUInt) loom.BigUInt {
	updatedAmount := loom.BigUInt{big.NewInt(billionthsBasisPointRatio)}
	updatedAmount.Mul(&updatedAmount, &bps)
	return updatedAmount
}

func calculateWeightedWhitelistAmount(statistic ValidatorStatistic) loom.BigUInt {
	// WhitelistLockTime must be 0, 1, 2, or 3. Any other value will be considered to give 5% rewards.
	tier, found := TierMap[statistic.WhitelistLocktime]
	if !found {
		tier = TIER_ZERO
	}
	return CalculateFraction(TierBonusMap[tier], statistic.WhitelistAmount.Value, true)
}

// LOGGING

func logDposError(ctx contract.Context, err error, req string) error {
	ctx.Logger().Error("DPOS error", "error", err, "sender", ctx.Message().Sender, "req", req)
	return err
}

func logStaticDposError(ctx contract.StaticContext, err error, req string) error {
	ctx.Logger().Error("DPOS static error", "error", err, "sender", ctx.Message().Sender, "req", req)
	return err
}
