package karma

import (
	"testing"

	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	lplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
)

func TestUpkeepParameters(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		lplugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}
	require.NoError(t, contract.Init(ctx, &ktypes.KarmaInitRequest{
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 3600,
		},
		Oracle: oracle,
		Users:  usersTestCoin,
	}))

	upkeep, err := contract.GetUpkeepParms(ctx, types_addr1)
	require.NoError(t, err)
	require.Equal(t, int64(1), upkeep.Cost)
	require.Equal(t, int64(3600), upkeep.Period)

	upkeep = &ktypes.KarmaUpkeepParams{
		Cost:   10,
		Period: 1000,
	}
	require.NoError(t, contract.SetUpkeepParams(ctx, upkeep))
	upkeep, err = contract.GetUpkeepParms(ctx, types_addr1)
	require.NoError(t, err)
	require.Equal(t, int64(10), upkeep.Cost)
	require.Equal(t, int64(1000), upkeep.Period)
}

func TestContractActivation(t *testing.T) {
	oracleAddr := addr1
	karmaInit := ktypes.KarmaInitRequest{
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 3600,
		},
		Oracle: oracleAddr.MarshalPB(),
		Users:  usersTestCoin,
	}

	fakeCtx := lplugin.CreateFakeContext(addr1, addr1)
	karmaAddr := fakeCtx.CreateContract(Contract)
	fakeCtx.RegisterContract("karma", karmaAddr, karmaAddr)

	ctx := contractpb.WrapPluginContext(fakeCtx.WithAddress(karmaAddr).WithSender(oracleAddr))

	karmaContract := &Karma{}
	require.NoError(t, karmaContract.Init(ctx, &karmaInit))

	// Mock EVM deploy Transaction
	evmContract := fakeCtx.CreateContract(nil)
	require.NoError(t, AddOwnedContract(ctx, addr1, evmContract))

	// Contract should've been activated when it was deployed
	isActive, err := IsContractActive(ctx, evmContract)
	require.NoError(t, err)
	require.True(t, isActive)

	records, err := GetActiveContractRecords(ctx, addr1)
	require.NoError(t, err)
	require.Len(t, records, 1)

	// Deactivate the contract and check contract status change propagates correctly
	require.NoError(t, karmaContract.SetInactive(ctx, evmContract.MarshalPB()))
	isActive, err = IsContractActive(ctx, evmContract)
	require.NoError(t, err)
	require.False(t, isActive)

	records, err = GetActiveContractRecords(ctx, addr1)
	require.NoError(t, err)
	require.Len(t, records, 0)

	users, err := GetActiveUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(users))

	// Reactivate the contract and check contract status change propagates correctly
	require.NoError(t, karmaContract.SetActive(ctx, evmContract.MarshalPB()))
	isActive, err = IsContractActive(ctx, evmContract)
	require.NoError(t, err)
	require.True(t, isActive)

	records, err = GetActiveContractRecords(ctx, addr1)
	require.NoError(t, err)
	require.Len(t, records, 1)

	users, err = GetActiveUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(users))
}
