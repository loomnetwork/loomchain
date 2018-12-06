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
		{"sms", 1, ktypes.KarmaSourceTarget_CALL},
		{"oauth", 3, ktypes.KarmaSourceTarget_CALL},
		{"token", 4, ktypes.KarmaSourceTarget_CALL},
		{DeployToken, 1, ktypes.KarmaSourceTarget_DEPLOY},
	}

	newSources = []*ktypes.KarmaSourceReward{
		{"token", 7, ktypes.KarmaSourceTarget_CALL},
		{"oauth", 2, ktypes.KarmaSourceTarget_CALL},
		{"new-call", 1, ktypes.KarmaSourceTarget_CALL},
		{"new-deploy", 3, ktypes.KarmaSourceTarget_DEPLOY},
		{DeployToken, 5, ktypes.KarmaSourceTarget_DEPLOY},
	}

	deploySource = []*ktypes.KarmaSourceReward{
		{DeployToken, 1, ktypes.KarmaSourceTarget_DEPLOY},
	}

	emptySourceStates = []*ktypes.KarmaSource{}

	sourceStates = []*ktypes.KarmaSource{
		{"sms", 1},
		{"oauth", 5},
		{"token", 10},
	}

	extremeSourceStates = []*ktypes.KarmaSource{
		{"sms", 1000},
		{"oauth", 5000},
		{"token", 10},
	}

	users = []*ktypes.KarmaAddressSource{
		{user, sourceStates},
		{oracle, sourceStates},
	}

	usersTestCoin= []*ktypes.KarmaAddressSource{
		{user, emptySourceStates},
		{oracle, emptySourceStates},
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

	s, err := contract.GetSources(ctx, &types.Address{})
	require.NoError(t, err)
	require.Equal(t, sources, s.Sources)
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


	err = contract.UpdateOracle(ctx2, &ktypes.KarmaNewOracleValidator{
		NewOracle: oracle2,
	})
	require.Error(t, err)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracleValidator{
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracleValidator{
		NewOracle: oracle,
	})
	require.Error(t, err)

	err = contract.UpdateOracle(ctx2, &ktypes.KarmaNewOracleValidator{
		NewOracle: oracle3,
	})
	require.NoError(t, err)
}

func TestKarmaUpkeep(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}
	require.NoError(t, contract.Init(ctx, &ktypes.KarmaInitRequest{
		Sources: deploySource,
		Oracle:  oracle,
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Source: "oauth",
			Period: 3600,
		},
	}))

	upkeep, err := contract.GetUpkeepParms(ctx, &types.Address{})
	require.NoError(t, err)
	require.Equal(t, int64(1), upkeep.Cost)
	require.Equal(t, "oauth",upkeep.Source)
	require.Equal(t, int64(3600), upkeep.Period)

	upkeep = &ktypes.KarmaUpkeepParams{
		Cost: 37,
		Source: "my source",
		Period: 1234,
	}
	require.NoError(t, contract.SetUpkeepParams(ctx, upkeep))

	upkeep, err = contract.GetUpkeepParms(ctx, &types.Address{})
	require.NoError(t, err)
	require.Equal(t, int64(37), upkeep.Cost)
	require.Equal(t, "my source",upkeep.Source)
	require.Equal(t, int64(1234), upkeep.Period)

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

	err = contract.DepositCoin(ctx, &ktypes.KarmaUserAmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(17)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(17), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(5)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(12), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(500)}})
	require.Error(t, err)
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
	require.Equal(t, extremeSourceStates, state.SourceStates)

	// GetUserState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User: user,
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
	require.Equal(t, []*ktypes.KarmaSource{{"token", 10}}, state.SourceStates)

	// GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetUserKarma(ctx,  &ktypes.KarmaUserTarget{
		User: user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(40), karmaTotal.Count)

	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracleValidator{
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	err = contract.ResetSources(ctx, &ktypes.KarmaSourcesValidator{
		Sources: newSources,
	})
	require.Error(t, err)

	ctx2 := contractpb.WrapPluginContext(fakeContext.WithSender(addr2))
	err = contract.ResetSources(ctx2, &ktypes.KarmaSourcesValidator{
		Sources: newSources,
	})
	require.NoError(t, err)

	karmaTotal, err = contract.GetUserKarma(ctx2,  &ktypes.KarmaUserTarget{
		User: user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(70), karmaTotal.Count)
}