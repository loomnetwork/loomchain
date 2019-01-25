package karma

import (
	"testing"

	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
)

func TestUpkeepParameters(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}
	require.NoError(t, contract.Init(ctx, &ktypes.KarmaInitRequest{
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 3600,
		},
		Oracle:  oracle,
		Users:   usersTestCoin,
	}))

	upkeep, err := contract.GetUpkeepParms(ctx, types_addr1)
	require.NoError(t, err)
	require.Equal(t, int64(1), upkeep.Cost )
	require.Equal(t, int64(3600), upkeep.Period )

	upkeep = &ktypes.KarmaUpkeepParams{
		Cost:   10,
		Period: 1000,
	}
	require.NoError(t, contract.SetUpkeepParams(ctx, upkeep))
	upkeep, err = contract.GetUpkeepParms(ctx, types_addr1)
	require.NoError(t, err)
	require.Equal(t, int64(10), upkeep.Cost )
	require.Equal(t, int64(1000), upkeep.Period )
}

func TestContractActivation(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 3600,
		},
		Oracle:  oracle,
		Users:   usersTestCoin,
	}

	state, reg, pluginVm := MockStateWithKarmaAndCoin(t, &karmaInit, nil, "mockAppDb1")
	karmaAddr, err := reg.Resolve("karma")
	require.NoError(t, err)
	ctx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, reg, addr3, karmaAddr, pluginVm),
	)

	karmaContract := &Karma{}

	// Mock Evm deploy Transaction
	block := int64(1)
	nonce := uint64(1)
	evmContract := MockDeployEvmContract(t, state, addr1, nonce, reg)
	karmaState := GetKarmaState(t, state, reg)
	require.NoError(t, AddOwnedContract(karmaState, addr1, evmContract, block, nonce))

	// Check consistency when toggling activation state
	activationState, err := karmaContract.IsActive(ctx, evmContract.MarshalPB())
	require.NoError(t, err)
	require.Equal(t, true, activationState)

	records, err := GetActiveContractRecords(karmaState)
	require.NoError(t, err)
	require.Len(t, records, 1)

	require.NoError(t, karmaContract.SetInactive(ctx, evmContract.MarshalPB()))
	activationState, err = karmaContract.IsActive(ctx, evmContract.MarshalPB())
	require.NoError(t, err)
	require.Equal(t, false, activationState)

	records, err = GetActiveContractRecords(karmaState)
	require.NoError(t, err)
	require.Len(t, records, 0)

	require.NoError(t, karmaContract.SetActive(ctx, evmContract.MarshalPB()))
	activationState, err = karmaContract.IsActive(ctx, evmContract.MarshalPB())
	require.NoError(t, err)
	require.Equal(t, true, activationState)

	records, err = GetActiveContractRecords(karmaState)
	require.NoError(t, err)
	require.Len(t, records, 1)
}
