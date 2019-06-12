package migrations

import "C"
import (
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/config"
	"github.com/pkg/errors"
)

var (
	InvalidBaseMintingAmount = errors.New("Base Minting Amount should be greater than zero")
)

type (
	Policy = ctypes.Policy
)

var (
	policyKey = []byte("policy")
)

func GenerateCoinPolicyMigrationFn(cfg *config.Config) func(ctx *MigrationContext) error {
	return func(ctx *MigrationContext) error {
		//Resolve coin context
		_, coinCtx, err := resolveCoin(ctx)
		if err != nil {
			return err
		}
		div := loom.NewBigUIntFromInt(10)
		div.Exp(div, loom.NewBigUIntFromInt(18), nil)
		if len(cfg.CoinPolicyMigrationConfig.MintingAccount) == 0 {
			return errors.New("Invalid Minting Account Address")
		}
		if cfg.CoinPolicyMigrationConfig.DeflationFactorNumerator <= 0 {
			return errors.New("DeflationFactorNumerator should be greater than zero")
		}
		if cfg.CoinPolicyMigrationConfig.DeflationFactorDenominator <= 0 {
			return errors.New("DeflationFactorDenominator should be greater than zero")
		}
		if cfg.CoinPolicyMigrationConfig.BaseMintingAmount <= 0 {
			return InvalidBaseMintingAmount
		}
		deflationFactorNumerator := cfg.CoinPolicyMigrationConfig.DeflationFactorNumerator
		deflationFactorDenominator := cfg.CoinPolicyMigrationConfig.DeflationFactorDenominator
		addr, err := loom.ParseAddress(cfg.CoinPolicyMigrationConfig.MintingAccount)
		if err != nil {
			return err
		}
		if cfg.CoinPolicyMigrationConfig.BaseMintingAmount <= 0 {
			return InvalidBaseMintingAmount
		}
		baseMintingAmount := loom.NewBigUIntFromInt(int64(cfg.CoinPolicyMigrationConfig.BaseMintingAmount))
		baseMintingAmount.Mul(baseMintingAmount, div)
		policy := &Policy{
			DeflationFactorNumerator:   deflationFactorNumerator,
			DeflationFactorDenominator: deflationFactorDenominator,
			BaseMintingAmount: &types.BigUInt{
				Value: *baseMintingAmount,
			},
			MintingAccount: addr.MarshalPB(),
		}
		err = coinCtx.Set(policyKey, policy)
		if err != nil {
			return err
		}

		// Turn on coin policy
		ctx.State().SetFeature(loomchain.CoinPolicyFeature, true)

		return nil
	}
}

func resolveCoin(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	coinCtx, err := ctx.ContractContext("coin")
	if err != nil {
		return loom.Address{}, nil, err
	}
	coinAddr := coinCtx.ContractAddress()
	return coinAddr, coinCtx, nil
}
