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
	addr4 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()
	types_addr4 = addr4.MarshalPB()


	oracle  = types_addr1
	oracle2 = types_addr2
	user    = types_addr4

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

	s, err := contract.GetSources(ctx, oracle)
	require.NoError(t, err)
	require.Equal(t, sources, s.Sources)
	for _, u := range users {
		require.True(t, ctx.Has(GetUserStateKey(u.User)))
		state, err := contract.GetUserState(ctx, u.User)
		require.NoError(t, err)
		require.Equal(t, len(sourceStates), len(state.SourceStates))
	}
}

func TestKarmaValidateOracle(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Oracle: oracle,
	})
	require.NoError(t, err)

	err = contract.validateOracle(ctx, oracle)
	require.NoError(t, err)

	err = contract.validateOracle(ctx, user)
	require.Error(t, err)

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
	require.Error(t, err)
	require.Equal(t, "not found", err.Error())

	err = contract.DepositCoin(ctx, &ktypes.KarmaUserAmmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(17)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(17), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(5)}})
	require.NoError(t, err)

	userState, err = contract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, DeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(12), userState.SourceStates[0].Count)

	err = contract.WithdrawCoin(ctx, &ktypes.KarmaUserAmmount{user, &types.BigUInt{*loom.NewBigUIntFromInt(500)}})
	require.Error(t, err)
}

func TestKarmaLifeCycleTest(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
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
		Oracle:       oracle,
		SourceStates: extremeSourceStates,
	})
	require.NoError(t, err)

	// GetUserState after UpdateSourcesForUser Test to test the change
	state, err := contract.GetUserState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, extremeSourceStates, state.SourceStates)

	// GetUserState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetTotal(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, int64(16040), karmaTotal.Count)

	// DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:      ko,
		Oracle:    oracle,
		StateKeys: deleteSourceKeys,
	})
	require.NoError(t, err)

	// GetUserState after DeleteSourcesForUser Test
	state, err = contract.GetUserState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, []*ktypes.KarmaSource{{"token", 10}}, state.SourceStates)

	// GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, int64(40), karmaTotal.Count)

	// Update entire config anf change oracle
	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracleValidator{
		OldOracle: oracle,
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	err = contract.ResetSources(ctx, &ktypes.KarmaSourcesValidator{
		Sources: newSources,
		Oracle: oracle2,
	})

	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, int64(70), karmaTotal.Count)
}
