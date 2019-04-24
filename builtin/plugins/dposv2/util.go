package dposv2

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

const billionthsBasisPointRatio = 100000

var (
	plasmaValidators = []loom.Address{
		loom.MustParseAddress("default:0x0e99fc16e32e568971908f2ce54b967a42663a26"), // plasma-0
		loom.MustParseAddress("default:0xac3211caecc45940a6d2ba006ca465a647d8464f"), // plasma-1
		loom.MustParseAddress("default:0x69c48768dbac492908161be787b7a5658192df35"), // plasma-2
		loom.MustParseAddress("default:0x2a3a7c850586d4f80a12ac1952f88b1b69ef48e1"), // plasma-3
		loom.MustParseAddress("default:0x4a1b8b15e50ce63cc6f65603ea79be09206cae70"), // plasma-4
		loom.MustParseAddress("default:0x0ce7b61c97a6d5083356f115288f9266553e191e"), // plasma-5
	}
	doubledDelegator = loom.MustParseAddress("default:0xDc93E46f6d22D47De9D7E6d26ce8c3b7A13d89Cb")
	doubledValidator = loom.MustParseAddress("default:0xa38c27e8cf4a443e805065065aefb250b1e1cef2")
	basisPoints      = loom.BigUInt{big.NewInt(1e4)} // do not change
	billionth        = loom.BigUInt{big.NewInt(1e9)}
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

/// If the validator is one of the plasma nodes, it sets it to plasma-0
func adjustValidatorIfInPlasmaValidators(delegation Delegation) *types.Address {
	validator := delegation.Validator
	for _, plasmaValidator := range plasmaValidators {
		if validator.Local.Compare(plasmaValidator.Local) == 0 {
			return plasmaValidators[0].MarshalPB()
		}
	}
	return validator
}

func adjustDoubledDelegationAmount(delegation Delegation) *types.BigUInt {
	amount := delegation.Amount.Value
	validatorMatch := doubledValidator.Local.Compare(delegation.Validator.Local) == 0
	delegatorMatch := doubledDelegator.Local.Compare(delegation.Delegator.Local) == 0
	if validatorMatch && delegatorMatch {
		amount = *common.BigZero()
		amount.Div(&delegation.Amount.Value, loom.NewBigUIntFromInt(2))
	}
	return &types.BigUInt{Value: amount}
}

// frac is expressed in basis points if granular == false
// or billionths if granular == true
func CalculateFraction(frac loom.BigUInt, total loom.BigUInt, granular bool) loom.BigUInt {
	if granular {
		frac = basisPointsToBillionths(frac)
	}
	return CalculatePreciseFraction(frac, total, granular)
}

func CalculatePreciseFraction(frac loom.BigUInt, total loom.BigUInt, granular bool) loom.BigUInt {
	denom := basisPoints
	if granular {
		denom = billionth
	}
	updatedAmount := *common.BigZero()
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

func calculateWeightedDelegationAmount(delegation Delegation, granular bool) loom.BigUInt {
	bonusPercentage := TierBonusMap[delegation.LocktimeTier]
	return CalculateFraction(bonusPercentage, delegation.Amount.Value, granular)
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
