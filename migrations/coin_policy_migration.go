package migrations

import (
	"errors"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

var (
	policyKey = []byte("policy")
)

//Passing Input Paramters as byte array to Coin Policy Migration Function
func GenerateCoinPolicyMigrationFn(ctx *MigrationContext, parameters []byte) error {
	//Resolve coin context
	_, coinCtx, err := resolveCoin(ctx)
	if err != nil {
		return err
	}
	coinPolicy := ctypes.Policy{}
	err = proto.Unmarshal(parameters, &coinPolicy)
	if err != nil {
		return err

	}
	if coinPolicy.ChangeRatioNumerator == 0 {
		return errors.New("ChangeRatioNumerator should be greater than zero")
	}
	if coinPolicy.ChangeRatioDenominator == 0 {
		return errors.New("ChangeRatioDenominator should be greater than zero")
	}
	if coinPolicy.BasePercentage == 0 {
		return errors.New("BasePercentage should be greater than zero")
	}
	if coinPolicy.TotalSupply == 0 {
		return errors.New("Total Supply should be greater than zero")
	}
	if coinPolicy.BlocksGeneratedPerYear == 0 {
		return errors.New("Blocks Generated Per Year should be greater than zero")
	}
	if !strings.EqualFold("div", coinPolicy.Operator) && !strings.EqualFold("exp", coinPolicy.Operator) {
		return errors.New("Invalid operator - Operator should be div or exp")
	}
	addr := loom.UnmarshalAddressPB(coinPolicy.MintingAccount)
	if addr.Compare(loom.RootAddress(addr.ChainID)) == 0 {
		return errors.New("Minting Account Address cannot be Root Address")
	}
	if err := coinCtx.Set(policyKey, &coinPolicy); err != nil {
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
