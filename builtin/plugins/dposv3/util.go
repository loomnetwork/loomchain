package dposv3

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
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

// frac is expressed in basis points
func CalculateFraction(frac loom.BigUInt, total loom.BigUInt) loom.BigUInt {
	updatedAmount := loom.BigUInt{big.NewInt(0)}
	updatedAmount.Mul(&total, &frac)
	updatedAmount.Div(&updatedAmount, &basisPoints)
	return updatedAmount
}

func calculateShare(delegation loom.BigUInt, total loom.BigUInt, rewards loom.BigUInt) loom.BigUInt {
	frac := loom.BigUInt{big.NewInt(0)}
	if (&total).Cmp(&frac) != 0 {
		frac.Mul(&delegation, &basisPoints)
		frac.Div(&frac, &total)
	}
	return CalculateFraction(frac, rewards)
}

func scientificNotation(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

func calculateWeightedDelegationAmount(delegation Delegation) loom.BigUInt {
	bonusPercentage := TierBonusMap[delegation.LocktimeTier]
	return CalculateFraction(bonusPercentage, delegation.Amount.Value)
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
