package karma

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()
	types_addr3 = addr3.MarshalPB()

	oracle  = types_addr1
	oracle2 = types_addr2
	oracle3 = types_addr3
	user    = types_addr3

	sources = []*ktypes.KarmaSourceReward{
		{ Name: "sms", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "oauth", Reward: 3, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "token", Reward: 4, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: DeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	newSources = []*ktypes.KarmaSourceReward{
		{ Name: "token", Reward: 7, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "oauth", Reward: 2, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "new-call", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "new-deploy", Reward: 3, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{ Name: DeployToken, Reward: 5, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	deploySource = []*ktypes.KarmaSourceReward{
		{ Name: DeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	emptySourceStates = []*ktypes.KarmaSource{}

	sourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: 1},
		{Name: "oauth", Count: 5},
		{Name: "token", Count: 10},
	}

	extremeSourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: 1000},
		{Name: "oauth", Count: 5000},
		{Name: "token", Count: 10},
	}

	users = []*ktypes.KarmaAddressSource{
		{User: user, Sources: sourceStates},
		{User: oracle, Sources: sourceStates},
	}

	usersTestCoin = []*ktypes.KarmaAddressSource{
		{ User: user, Sources: emptySourceStates},
		{ User: oracle, Sources:  emptySourceStates},
	}

	deleteSourceKeys = []string{"sms", "oauth"}
)

func TestKarmaInit(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Oracle:  oracle,
		Sources: sources,
		Users:   users,
	})
	require.Nil(t, err)

	s, err := contract.GetSources(ctx, oracle)
	require.NoError(t, err)
	for k := range sources {
		require.Equal(t, sources[k].String(), s.Sources[k].String())
	}
	for _, u := range users {
		require.True(t, ctx.Has(UserStateKey(u.User)))
		state, err := contract.GetUserState(ctx, u.User)
		require.NoError(t, err)
		require.Equal(t, len(sourceStates), len(state.SourceStates))
	}
}

func TestKarmaValidateOracle(t *testing.T) {
	fakeContext := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(fakeContext)
	ctx2 := contractpb.WrapPluginContext(fakeContext.WithSender(addr2))

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Oracle: oracle,
	})
	require.NoError(t, err)

	err = contract.UpdateOracle(ctx2, &ktypes.KarmaNewOracle{
		NewOracle: oracle2,
	})
	require.Error(t, err)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracle{
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracle{
		NewOracle: oracle,
	})
	require.Error(t, err)

	err = contract.UpdateOracle(ctx2, &ktypes.KarmaNewOracle{
		NewOracle: oracle3,
	})
	require.NoError(t, err)
}

func TestKarmaCoin(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}
	require.NoError(t, contract.Init(ctx, &ktypes.KarmaInitRequest{
		Sources: deploySource,
		Oracle:  oracle,
		Users:   usersTestCoin,
	}))

	userState, err := contract.GetUserState(ctx, user)
	require.NoError(t, err)

	err = contract.DepositCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(17)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(17), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(12), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(500)}})
	require.Error(t, err)
}

func TestUpkeepParameters(t *testing.T) {
    ctx := contractpb.WrapPluginContext(
        plugin.CreateFakeContext(addr1, addr1),
    )
    contract := &Karma{}
    require.NoError(t, contract.Init(ctx, &ktypes.KarmaInitRequest{
        Sources: []*ktypes.KarmaSourceReward{
            {Name: DeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
        },
        Upkeep: &ktypes.KarmaUpkeepParams{
            Cost:   1,
            Source: DeployToken,
            Period: 3600,
        },
        Oracle:  oracle,
        Users:   usersTestCoin,
    }))

    upkeep, err := contract.GetUpkeepParms(ctx, types_addr1)
    require.NoError(t, err)
    require.Equal(t, int64(1), upkeep.Cost )
    require.Equal(t, DeployToken, upkeep.Source )
    require.Equal(t, int64(3600), upkeep.Period )

    upkeep = &ktypes.KarmaUpkeepParams{
        Cost:   10,
        Source: "TestDeploy",
        Period: 1000,
    }
    require.NoError(t, contract.SetUpkeepParams(ctx, upkeep))
    upkeep, err = contract.GetUpkeepParms(ctx, types_addr1)
    require.NoError(t, err)
    require.Equal(t, int64(10), upkeep.Cost )
    require.Equal(t, "TestDeploy", upkeep.Source )
    require.Equal(t, int64(1000), upkeep.Period )
}

func TestContractActivation(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Sources: []*ktypes.KarmaSourceReward{
			{Name: DeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		},
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Source: DeployToken,
			Period: 3600,
		},
		Oracle:  oracle,
		Users:   usersTestCoin,
	}
	state := MockStateWithKarma(t, karmaInit)
	karmaAddr := GetKarmaAddress(t, state)
	ctx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, addr1, karmaAddr),
	)

	karmaContract := &Karma{}

	// Mock Evm deploy Transaction
	block := int64(1)
	nonce := uint64(1)
	evmContract := MockDeployEvmContract(t, state, addr1, nonce)
	karmaState := GetKarmaState(t, state)
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

func TestKarmaLifeCycleTest(t *testing.T) {
	fakeContext := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(fakeContext)

	contract := &Karma{}
	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Sources: sources,
		Oracle:  oracle,
	})
	require.NoError(t, err)
	ko := user

	// UpdateSourcesForUser Test
	err = contract.AppendSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:         ko,
		SourceStates: extremeSourceStates,
	})
	require.NoError(t, err)

	// GetUserState after UpdateSourcesForUser Test to test the change
	state, err := contract.GetUserState(ctx, ko)
	require.NoError(t, err)
	for k := range extremeSourceStates {
		require.Equal(t, extremeSourceStates[k].String(), state.SourceStates[k].String())
	}

	// GetUserState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(16040), karmaTotal.Count)

	// DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:      ko,
		StateKeys: deleteSourceKeys,
	})
	require.NoError(t, err)

	// GetUserState after DeleteSourcesForUser Test
	state, err = contract.GetUserState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, []*ktypes.KarmaSource{{Name: "token", Count: 10}}, state.SourceStates)

	// GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(40), karmaTotal.Count)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracle{
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	err = contract.ResetSources(ctx, &ktypes.KarmaSources{
		Sources: newSources,
	})
	require.Error(t, err)

	ctx2 := contractpb.WrapPluginContext(fakeContext.WithSender(addr2))
	err = contract.ResetSources(ctx2, &ktypes.KarmaSources{
		Sources: newSources,
	})
	require.NoError(t, err)

	karmaTotal, err = contract.GetUserKarma(ctx2, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(70), karmaTotal.Count)
}

