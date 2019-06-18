package migrations

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

var (
	policyKey = []byte("policy")
)

type (
	Policy = ctypes.Policy
)

func GenerateCoinPolicyMigrationFn(ctx *MigrationContext, parameters []byte) error {
	//Resolve coin context
	_, coinCtx, err := resolveCoin(ctx)
	if err != nil {
		return err
	}
	coinPolicy := Policy{}
	err = proto.Unmarshal([]byte(parameters), &coinPolicy)
	if err != nil {
		return err

	}
	if coinPolicy.DeflationFactorNumerator == 0 {
		return errors.New("DeflationFactorNumerator should be greater than zero")
	}

	if coinPolicy.DeflationFactorDenominator == 0 {
		return errors.New("DeflationFactorDenominator should be greater than zero")
	}
	if coinPolicy.BaseMintingAmount == 0 {
		return errors.New("Base Minting Amount should be greater than zero")
	}
	addr := loom.UnmarshalAddressPB(coinPolicy.MintingAccount)

	if addr.Compare(loom.RootAddress(addr.ChainID)) == 0 {
		return errors.New("Minting Account Address cannot be Root Address")
	}
	err = coinCtx.Set(policyKey, &coinPolicy)
	if err != nil {
		return err
	}
	// Turn on coin policy
	ctx.State().SetFeature(loomchain.CoinVersion1_2Feature, true)
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
