package migrations

import (
	"encoding/json"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/config"
)

const (
	featurePrefix = "ft"
)

func featureKey(featureName string) []byte {
	return util.PrefixKey([]byte(featurePrefix), []byte(featureName))
}

func DPOSv3Migration(ctx *MigrationContext) error {
	// Pull data from DPOSv2
	_, dposv2Ctx, err := resolveDPOSv2(ctx)
	if err != nil {
		return err
	}

	_, chainconfigCtx, err := resolveChainConfig(ctx)
	if err != nil {
		return err
	}

	// This init information is ignored because dposv3.Initialize resets all
	// contract storage. However, ctx.DeployContract requires making a dummy
	// call to dposv3.Init initially.
	initRequest := dposv3.InitRequest{
		Params: &dposv3.Params{},
	}
	init, err := json.Marshal(initRequest)
	if err != nil {
		return err
	}
	contractConfig := config.ContractConfig{
		VMTypeName: "plugin",
		Format:     "plugin",
		Name:       "dposV3",
		Location:   "dposV3:3.0.0",
		Init:       init,
	}
	dposv3Addr, err := ctx.DeployContract(&contractConfig)
	if err != nil {
		return err
	}

	// Dump dposv2 state into a v3-compatible form and transfer dposv2 balance
	// to dposv3
	initializationState, err := dposv2.Dump(dposv2Ctx, dposv3Addr)
	if err != nil {
		return err
	}

	dposv3Ctx, err := ctx.ContractContext("dposV3")
	if err != nil {
		return err
	}
	dposv3.Initialize(dposv3Ctx, initializationState)

	// Set feature in chainconfig Contract
	if err := chainconfig.AddSpecificFeature(chainconfigCtx,string(featureKey(loomchain.DPOSVersion3Feature)),uint64(chainconfigCtx.Block().Height)); err != nil {
		return err
	}
	// Switch over to DPOSv3
	ctx.State().SetFeature(loomchain.DPOSVersion3Feature, true)
	return nil
}

func resolveDPOSv2(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	dposv2Ctx, err := ctx.ContractContext("dposV2")
	if err != nil {
		return loom.Address{}, nil, err
	}
	dposv2Addr := dposv2Ctx.ContractAddress()
	return dposv2Addr, dposv2Ctx, nil
}

func resolveChainConfig(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	chainconfigCtx, err := ctx.ContractContext("chainconfig")
	if err != nil {
		return loom.Address{}, nil, err
	}
	chainconfigAddr := chainconfigCtx.ContractAddress()
	return chainconfigAddr, chainconfigCtx, nil
}