package karma

import (
	"fmt"
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/stretchr/testify/require"
)

var (
	addr1     = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2     = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3     = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	user_addr = addr3

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()
	types_addr3 = addr3.MarshalPB()

	oracle  = types_addr1
	oracle2 = types_addr2
	oracle3 = types_addr3
	user    = types_addr3

	sources = []*ktypes.KarmaSourceReward{
		{Name: "sms", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "oauth", Reward: 3, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "token", Reward: 4, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	newSources = []*ktypes.KarmaSourceReward{
		{Name: "token", Reward: 7, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "oauth", Reward: 2, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "new-call", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "new-deploy", Reward: 3, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: CoinDeployToken, Reward: 5, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	deploySource = []*ktypes.KarmaSourceReward{
		{Name: CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	emptySourceStates = []*ktypes.KarmaSource{}

	sourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)}},
		{Name: "oauth", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}},
		{Name: "token", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}},
	}

	extremeSourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1000)}},
		{Name: "oauth", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5000)}},
		{Name: "token", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}},
	}

	users = []*ktypes.KarmaAddressSource{
		{User: user, Sources: sourceStates},
		{User: oracle, Sources: sourceStates},
	}

	usersTestCoin = []*ktypes.KarmaAddressSource{
		{User: user, Sources: emptySourceStates},
		{User: oracle, Sources: emptySourceStates},
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
		Config:  &ktypes.KarmaConfig{MinKarmaToDeploy: 73},
	})
	require.Nil(t, err)

	s, err := contract.GetSources(ctx, &ktypes.GetSourceRequest{})
	require.NoError(t, err)
	for k := range sources {
		require.Equal(t, sources[k].String(), s.Sources[k].String())
	}
	for _, u := range users {
		key, err := UserStateKey(u.User)
		require.NoError(t, err)
		require.True(t, ctx.Has(key))
		state, err := contract.GetUserState(ctx, u.User)
		require.NoError(t, err)
		require.Equal(t, len(sourceStates), len(state.SourceStates))
	}
	config, err := contract.GetConfig(ctx, &ktypes.GetConfigRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(73), config.MinKarmaToDeploy)

	require.NoError(t, contract.SetConfig(ctx, &ktypes.KarmaConfig{MinKarmaToDeploy: 85}))

	config, err = contract.GetConfig(ctx, &ktypes.GetConfigRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(85), config.MinKarmaToDeploy)
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
	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Oracle:  oracle,
		Users:   usersTestCoin,
	}

	coinInit := coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user, Balance: uint64(100)},
		},
	}

	state, reg, pluginVm := MockStateWithKarmaAndCoinT(t, &karmaInit, &coinInit)
	karmaAddr, err := reg.Resolve("karma")
	require.NoError(t, err)
	ctx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, reg, addr3, karmaAddr, pluginVm),
	)
	karmaContract := &Karma{}

	coinAddr, err := reg.Resolve("coin")
	require.NoError(t, err)
	coinContract := &coin.Coin{}
	coinCtx := contractpb.WrapPluginContext(
		CreateFakeStateContext(state, reg, user_addr, coinAddr, pluginVm),
	)
	approveRequest := &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
	}
	fmt.Printf("Approve Request: %+v\n", approveRequest)
	err = coinContract.Approve(coinCtx, approveRequest)
	fmt.Println(err)
	require.NoError(t, err)
	fmt.Println("Karma coin address", karmaAddr.String())
	allowanceRequest := &coin.AllowanceRequest{
		Owner:   user,
		Spender: karmaAddr.MarshalPB(),
	}
	res, err := coinContract.Allowance(coinCtx, allowanceRequest)
	require.NoError(t, err)
	fmt.Println("Karma contract allowance", res.Amount.Value.String())

	initalBal, err := coinContract.BalanceOf(coinCtx, &coin.BalanceOfRequest{Owner: user})
	fmt.Println("User balance", initalBal.Balance.String())
	require.NoError(t, err)

	userState, err := karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)

	fmt.Printf("Allowance Request: %+v\n", allowanceRequest)
	res, err = coinContract.Allowance(coinCtx, allowanceRequest)
	require.NoError(t, err)
	fmt.Println("Karma contract allowance", res.Amount.Value.String())

	err = karmaContract.DepositCoin(ctx, &ktypes.KarmaUserAmount{User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(17)}})
	require.NoError(t, err)
	balAfterDeposit, err := coinContract.BalanceOf(coinCtx, &coin.BalanceOfRequest{Owner: user})
	require.NoError(t, err)
	expected := common.BigZero()
	expected = expected.Sub(&initalBal.Balance.Value, loom.NewBigUIntFromInt(17))
	require.Equal(t, 0, expected.Cmp(&balAfterDeposit.Balance.Value))

	userState, err = karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, CoinDeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(17), userState.SourceStates[0].Count.Value.Int64())

	err = karmaContract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}})
	require.NoError(t, err)
	balAfterWithdrawal, err := coinContract.BalanceOf(coinCtx, &coin.BalanceOfRequest{Owner: user})
	require.NoError(t, err)
	expected = expected.Sub(&initalBal.Balance.Value, loom.NewBigUIntFromInt(17-5))
	require.Equal(t, 0, expected.Cmp(&balAfterWithdrawal.Balance.Value))

	userState, err = karmaContract.GetUserState(ctx, user)
	require.NoError(t, err)
	require.Equal(t, 1, len(userState.SourceStates))
	require.Equal(t, CoinDeployToken, userState.SourceStates[0].Name)
	require.Equal(t, int64(12), userState.SourceStates[0].Count.Value.Int64())

	total, err := karmaContract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_DEPLOY,
	})
	require.NoError(t, err)
	total = total

	err = karmaContract.WithdrawCoin(ctx, &ktypes.KarmaUserAmount{User: user, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(500)}})
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
	err = contract.AddKarma(ctx, &ktypes.AddKarmaRequest{
		User:         ko,
		KarmaSources: extremeSourceStates,
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
		Target: ktypes.KarmaSourceTarget_CALL,
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
	require.Equal(t, []*ktypes.KarmaSource{{Name: "token", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}}}, state.SourceStates)

	// GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetUserKarma(ctx, &ktypes.KarmaUserTarget{
		User:   user,
		Target: ktypes.KarmaSourceTarget_CALL,
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
		Target: ktypes.KarmaSourceTarget_CALL,
	})
	require.NoError(t, err)
	require.Equal(t, int64(70), karmaTotal.Count.Value.Int64())
}
