package migrations

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
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
	_, chainconfigCtx, err := resolveChainConfig(ctx)
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
	if coinPolicy.TotalSupply == 0 {
		return errors.New("Total Supply should be greater than zero")
	}
	if coinPolicy.BlocksGeneratedPerYear == 0 {
		return errors.New("Blocks Generated Per Year should be greater than zero")
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
	// Set feature in chainconfig Contract
	var feature cctypes.Feature
	feature.Status = cctypes.Feature_ENABLED
	feature.BlockHeight = uint64(chainconfigCtx.Block().Height)
	if err := chainconfigCtx.Set(featureKey(loomchain.CoinVersion1_2Feature), &feature); err != nil {
		return err
	}
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

func resolveChainConfig(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	chainconfigCtx, err := ctx.ContractContext("chainconfig")
	if err != nil {
		return loom.Address{}, nil, err
	}
	chainconfigAddr := chainconfigCtx.ContractAddress()
	return chainconfigAddr, chainconfigCtx, nil
}
