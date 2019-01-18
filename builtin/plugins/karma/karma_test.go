package karma

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	//ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	karmaAddr = loom.MustParseAddress("chain:0x7E402be3d3A83FF850dc22775d33E89fFD374dD1")

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
		{ Name: CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	newSources = []*ktypes.KarmaSourceReward{
		{ Name: "token", Reward: 7, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "oauth", Reward: 2, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "new-call", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{ Name: "new-deploy", Reward: 3, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{ Name: CoinDeployToken, Reward: 5, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	deploySource = []*ktypes.KarmaSourceReward{
		{ Name: CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	emptySourceStates = []*ktypes.KarmaSource{}

	sourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(1) } },
		{Name: "oauth", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(5) }},
		{Name: "token", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(10) }},
	}

	extremeSourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(1000) }},
		{Name: "oauth", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(5000) }},
		{Name: "token", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(10) }},
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
	require.Equal(t, AwardDeployToken, s.Sources[len(s.Sources)-1].Name)
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
	t.Skip("still working on test")
	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Oracle:  oracle,
		Users:   usersTestCoin,
	}

	coinInit := coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{ Owner:   user,	Balance: uint64(100) },
			{ Owner:   types_addr1,	Balance: uint64(100) },
			{ Owner:   types_addr2,	Balance: uint64(100) },
		},
	}

	state, reg, pluginVm := MockStateWithKarmaAndCoin(t, karmaInit, coinInit)
	karmaAddr := GetKarmaAddress(t, state)
	ctx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, reg, addr3, karmaAddr, pluginVm),
	)
	karmaContract := &Karma{}

	coinAddr, err := reg.Resolve("coin")
	coinContract := &coin.Coin{}
	coinCtx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, reg, addr3, coinAddr, pluginVm),
	)

	/*
	pctx := plugin.CreateFakeContext(addr1, karmaAddr)
	coinAddr := pctx.CreateContract(coin.Contract)
	pctx.RegisterContract("coin", coinAddr, coinAddr)
	ctx := contractpb.WrapPluginContext(pctx)

	karmaContract := &Karma{}
	require.NoError(t, karmaContract.Init(ctx, &karmaInit))
	coinContract := &coin.Coin{}
	require.NoError(t, coinContract.Init(ctx, &coinInit))
	*/

	require.NoError(t,coinContract.Approve(coinCtx, &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),// ctx.ContractAddress().MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
	}))

	userState, err := karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)

	err = karmaContract.DepositCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(17)}})
	require.NoError(t, err)

	userState, err = karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, CoinDeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(17), userState.SourceStates[0].Count.Value.Int64())

	err = karmaContract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}})
	require.NoError(t, err)

	userState, err = karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, CoinDeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(12), userState.SourceStates[0].Count)

	total, err := karmaContract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,

	})
	require.NoError(t, err)
	total = total

	err = karmaContract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{ User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(500)}})
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
	for k := range extremeSourceStates {
		require.Equal(t, extremeSourceStates[k].Name, state.SourceStates[k].Name)
		require.Equal(t, 0, extremeSourceStates[k].Count.Value.Cmp(&state.SourceStates[k].Count.Value))
	}

	// GetUserState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(16040), karmaTotal.Count.Value.Int64())

	// DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:      ko,
		StateKeys: deleteSourceKeys,
	})
	require.NoError(t, err)

	// GetUserState after DeleteSourcesForUser Test
	state, err = contract.GetUserState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, []*ktypes.KarmaSource{{Name: "token", Count: &types.BigUInt{ Value: *loom.NewBigUIntFromInt(10) }}}, state.SourceStates)

	// GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_ALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(40), karmaTotal.Count.Value.Int64())

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
	require.Equal(t, int64(70), karmaTotal.Count.Value.Int64())
}
