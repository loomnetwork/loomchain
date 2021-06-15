package vm

import (
	"math/big"

	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/pkg/errors"
)

var (
	ErrGasTrackerOutOfGas = errors.New("[GasTracker] out of gas")
)

// GasTracker handles gas purchase, use, and refund for a single tx.
type GasTracker interface {
	ApproveGasPurchase(caller loom.Address, gas uint64, price *big.Int) error
	BuyGas(caller loom.Address, gas uint64, price *big.Int) error
	UseGas(gas uint64) error
	RemainingGas() uint64
	RefundGas(caller loom.Address)
}

type GasTrackerFactoryFunc func() (GasTracker, error)

// LegacyGasTracker doesn't charge anything for gas, and always allocates a fixed amount of gas
// (the max per-tx limit, regardless of how much is actually requested).
type LegacyGasTracker struct {
	gas uint64
}

func (gt *LegacyGasTracker) ApproveGasPurchase(_ loom.Address, _ uint64, _ *big.Int) error {
	return nil
}

func (gt *LegacyGasTracker) BuyGas(caller loom.Address, _ uint64, _ *big.Int) error {
	gt.gas = 0 // TODO: set this from the on-chain config
	return nil
}

func (gt *LegacyGasTracker) UseGas(gas uint64) error {
	if gas > gt.gas {
		return ErrGasTrackerOutOfGas
	}
	gt.gas -= gas
	return nil
}

func (gt *LegacyGasTracker) RemainingGas() uint64 {
	return gt.gas
}

func (gt *LegacyGasTracker) RefundGas(caller loom.Address) {
	// didn't actually buy the gas, so nothing to refund
}

// LoomCoinGasTracker charges LOOM for gas.
type LoomCoinGasTracker struct {
	coinCtx      contract.Context
	feeCollector loom.Address
	maxGas       uint64
	minPrice     *big.Int
	gas          uint64   // amount of gas remaining
	price        *big.Int // price the gas was purchased at
}

func NewLoomCoinGasTracker(
	coinCtx contract.Context, feeCollector loom.Address, maxGas uint64, minPrice *big.Int,
) *LoomCoinGasTracker {
	return &LoomCoinGasTracker{
		coinCtx:      coinCtx,
		feeCollector: feeCollector,
		maxGas:       maxGas,
		minPrice:     minPrice,
	}
}

func (gt *LoomCoinGasTracker) ApproveGasPurchase(caller loom.Address, gas uint64, price *big.Int) error {
	if price == nil || price.Sign() <= 0 {
		return errors.New("[GasTracker] gas price not specified")
	}

	if price.Cmp(gt.minPrice) < 0 {
		return errors.New("[GasTracker] gas price too low")
	}

	if gas > gt.maxGas {
		return errors.New("[GasTracker] gas exceeds per-tx limit")
	}

	// TODO: Ensure gas price specified by the caller matches the price set via the on-chain config.
	//       Later on the minimal price should be set via loom.yml, and the check here should
	//       simply ensure the caller price is not less than that (this will allow validators to
	//       adjust the price by consensus).

	// ensure caller has enough LOOM to cover the max gas they're willing to pay for
	fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), price)
	balance, err := coin.BalanceOf(gt.coinCtx, caller)
	if err != nil {
		return errors.New("[GasTracker] failed to check gas buyer balance")
	}
	if fee.Cmp(balance.Int) > 0 {
		return errors.New("[GasTracker] gas fee exceeds available balance")
	}
	return nil
}

func (gt *LoomCoinGasTracker) BuyGas(caller loom.Address, gas uint64, price *big.Int) error {
	// assumption: both gas and price parameters have been validated by ApproveGasPurchase already
	fee := loom.NewBigUInt(new(big.Int).Mul(new(big.Int).SetUint64(gas), price))
	if err := coin.Transfer(gt.coinCtx, caller, gt.feeCollector, fee); err != nil {
		return errors.Wrap(err, "[GasTracker] failed to transfer fee from caller")
	}

	gt.gas = gas
	gt.price = price
	return nil
}

func (gt *LoomCoinGasTracker) UseGas(gas uint64) error {
	if gas > gt.gas {
		return ErrGasTrackerOutOfGas
	}
	gt.gas -= gas
	return nil
}

func (gt *LoomCoinGasTracker) RemainingGas() uint64 {
	return gt.gas
}

func (gt *LoomCoinGasTracker) RefundGas(caller loom.Address) {
	// TODO: handle error
	coin.Transfer(gt.coinCtx, gt.feeCollector, caller, loom.NewBigUInt(new(big.Int).SetUint64(gt.gas)))
}
