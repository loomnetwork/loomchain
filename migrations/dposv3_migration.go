package migrations

import (
	"encoding/json"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/config"

	dposv2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	dposv3types "github.com/loomnetwork/go-loom/builtin/types/dposv3"
)

func DPOSv3Migration(ctx *MigrationContext) error {
	// Pull data from DPOSv2
	dposv2Addr, dposv2Ctx, err := resolveDPOSv2(ctx)
	if err != nil {
		return err
	}

	// Direct call
	dposv2.ValidatorList(dposv2Ctx)

	// Indirect call
	listReqV2 := &dposv2types.ListValidatorsRequestV2{}
	var listRespV2 dposv2types.ListValidatorsResponseV2
	contractpb.StaticCallMethod(dposv2Ctx, dposv2Addr, "ListValidators", listReqV2, &listRespV2)

	// Deploy DPOS v3 with v2 data
	dposv3Addr, dposv3Ctx, err := deployDPOSv3(ctx)
	if err != nil {
		return err
	}

	dposv3.ValidatorList(dposv3Ctx)

	listReqV3 := &dposv3types.ListValidatorsRequest{}
	var listRespV3 dposv3types.ListValidatorsResponse
	contractpb.StaticCallMethod(dposv3Ctx, dposv3Addr, "ListValidators", listReqV3, &listRespV3)

	// Switch over to DPOSv3
	ctx.State().SetFeature(loomchain.DPOSVersion3Feature, true)

	return nil
}

func deployDPOSv3(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	// TODO: populate initRequest with data from DPOSv2
	oracleAddr := loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	initRequest := dposv3.InitRequest{
		Params: &dposv3.Params{
			ValidatorCount: 21,
			OracleAddress:  oracleAddr.MarshalPB(),
		},
		Validators: []*dposv3.Validator{
			&dposv3.Validator{
				PubKey: []byte("IEcXesXZUwaDjTndcS751JybWYZtH2IbivTWBnDvyNI="),
				Power:  10,
			},
		},
	}
	init, err := json.Marshal(initRequest)
	if err != nil {
		return loom.Address{}, nil, err
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
		return loom.Address{}, nil, err
	}
	dposv3Ctx, err := ctx.ContractContext("dposV3")
	if err != nil {
		return loom.Address{}, nil, err
	}

	return dposv3Addr, dposv3Ctx, nil
}

func resolveDPOSv2(ctx *MigrationContext) (loom.Address, contractpb.Context, error) {
	dposv2Ctx, err := ctx.ContractContext("dposV2")
	if err != nil {
		return loom.Address{}, nil, err
	}
	dposv2Addr := dposv2Ctx.ContractAddress()
	return dposv2Addr, dposv2Ctx, nil
}
