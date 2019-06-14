package migrations

import "C"
import (
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/config"
	"github.com/pkg/errors"
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
		if cfg.CoinPolicyMigration == nil {
			return errors.New("Coin Policy not specified")
		}
		err = cfg.CoinPolicyMigration.IsValid()
		if err != nil {
			return err
		}
		addr, _ := loom.ParseAddress(cfg.CoinPolicyMigration.MintingAccount)
		deflationFactorNumerator := cfg.CoinPolicyMigration.DeflationFactorNumerator
		deflationFactorDenominator := cfg.CoinPolicyMigration.DeflationFactorDenominator
		baseMintingAmount := cfg.CoinPolicyMigration.BaseMintingAmount
		policy := &Policy{
			DeflationFactorNumerator:   deflationFactorNumerator,
			DeflationFactorDenominator: deflationFactorDenominator,
			BaseMintingAmount:          baseMintingAmount,
			MintingAccount:             addr.MarshalPB(),
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
