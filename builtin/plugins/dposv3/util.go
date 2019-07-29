package dposv3

import (
	"errors"
	"fmt"
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

// frac is expressed in basis points
func CalculateFraction(frac loom.BigUInt, total loom.BigUInt) loom.BigUInt {
	return CalculatePreciseFraction(basisPointsToBillionths(frac), total)
}

// frac is expressed in billionths
func CalculatePreciseFraction(frac loom.BigUInt, total loom.BigUInt) loom.BigUInt {
	updatedAmount := *common.BigZero()
	updatedAmount.Mul(&total, &frac)
	updatedAmount.Div(&updatedAmount, &billionth)
	return updatedAmount
}

func calculateShare(delegation loom.BigUInt, total loom.BigUInt, rewards loom.BigUInt) loom.BigUInt {
	frac := common.BigZero()
	if !common.IsZero(total) {
		frac.Mul(&delegation, &billionth)
		frac.Div(frac, &total)
	}
	return CalculatePreciseFraction(*frac, rewards)
}

func scientificNotation(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

// Locktime Tiers are enforced to be 0-3 for 5-20% rewards. Any other number is reset to 5%. We add the check just in case somehow the variable gets misset.
func calculateWeightedDelegationAmount(delegation Delegation) loom.BigUInt {
	bonusPercentage, found := TierBonusMap[delegation.LocktimeTier]
	if !found {
		bonusPercentage = TierBonusMap[TIER_ZERO]
	}
	return CalculateFraction(bonusPercentage, delegation.Amount.Value)
}

// Locktime Tiers are enforced to be 0-3 for 5-20% rewards. Any other number is reset to 5%. We add the check just in case somehow the variable gets misset.
func calculateWeightedWhitelistAmount(statistic ValidatorStatistic) loom.BigUInt {
	bonusPercentage, found := TierBonusMap[statistic.LocktimeTier]
	if !found {
		bonusPercentage = TierBonusMap[TIER_ZERO]
	}
	return CalculateFraction(bonusPercentage, statistic.WhitelistAmount.Value)
}

func basisPointsToBillionths(bps loom.BigUInt) loom.BigUInt {
	updatedAmount := loom.BigUInt{big.NewInt(billionthsBasisPointRatio)}
	updatedAmount.Mul(&updatedAmount, &bps)
	return updatedAmount
}

// VALIDATION

func validateCandidateFee(ctx contract.Context, fee uint64) error {
	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	if fee < state.Params.MinCandidateFee {
		return errors.New(fmt.Sprintf("Candidate Fee percentage cannot be less than %d", state.Params.MinCandidateFee))
	}

	return validateFee(fee)
}

func validateFee(fee uint64) error {
	if fee > hundredPercentInBasisPoints {
		return errors.New("Fee percentage cannot be greater than 100%.")
	}

	return nil
}

func validatePercentage(percentageBig loom.BigUInt) error {
	percentage := percentageBig.Int.Int64()
	if percentage < 0 || percentage > hundredPercentInBasisPoints {
		return errors.New(fmt.Sprintf("percentages must be from 0 - 10000 basis points"))
	}

	return nil
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

// LIMBO VALIDATOR

func LimboValidatorAddress(ctx contract.StaticContext) loom.Address {
	return loom.RootAddress(ctx.Block().ChainID)
}
