package migrations

import (
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	goloomvm "github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/pkg/errors"
)

type (
	Policy = ctypes.Policy
)

var (
	policyKey = []byte("policy")
)

func GenerateCoinPolicyMigrationFn(ctx *MigrationContext, parameters *goloomvm.MigrationParameters) error {
	//Resolve coin context
	_, coinCtx, err := resolveCoin(ctx)
	if err != nil {
		return err
	}
	if parameters.DeflationFactorNumerator == 0 {
		return errors.New("DeflationFactorNumerator should be greater than zero")
	}
	if parameters.DeflationFactorDenominator == 0 {
		return errors.New("DeflationFactorDenominator should be greater than zero")
	}
	if parameters.BaseMintingAmount == 0 {
		return errors.New("Base Minting Amount should be greater than zero")
	}
	addr, err := loom.ParseAddress(parameters.MintingAccount)
	if err != nil {
		return err
	}
	if addr.Compare(loom.RootAddress(addr.ChainID)) == 0 {
		return errors.New("Minting Account Address cannot be Root Address")
	}
	deflationFactorNumerator := parameters.DeflationFactorNumerator
	deflationFactorDenominator := parameters.DeflationFactorDenominator
	baseMintingAmount := parameters.BaseMintingAmount
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

func resolveCoin(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	coinCtx, err := ctx.ContractContext("coin")
	if err != nil {
		return loom.Address{}, nil, err
	}
	coinAddr := coinCtx.ContractAddress()
	return coinAddr, coinCtx, nil
}
