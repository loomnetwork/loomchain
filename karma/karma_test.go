package karma

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	lplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/stretchr/testify/require"
)

const (
	period = 3600
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()

	user1 = types_addr1
	user2 = types_addr2

	awardSoures = []*ktypes.KarmaSourceReward{
		{Name: "award1", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "award2", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "award3", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "award4", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	awardsSoures = []*ktypes.KarmaSource{
		{Name: "award1", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(21)}},
		{Name: "award2", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)}},
		{Name: "award3", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(17)}},
		{Name: "award4", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(11)}},
	}

	emptySourceStates = []*ktypes.KarmaSource{}
)

func TestAwardUpkeep(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Users:   []*ktypes.KarmaAddressSource{{User: user1, Sources: emptySourceStates}},
		Sources: awardSoures,
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   10,
			Period: period,
		},
		Oracle: user1,
	}
	coinInit := coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user1, Balance: uint64(200)},
			{Owner: user2, Balance: uint64(200)},
		},
	}

	fakeCtx := lplugin.CreateFakeContext(addr1, addr1)
	block := fakeCtx.Block()
	block.Height = 1
	fakeCtx = fakeCtx.WithBlock(block)

	karmaAddr := fakeCtx.CreateContract(karma.Contract)
	fakeCtx.RegisterContract("karma", karmaAddr, karmaAddr)
	karmaContract := &karma.Karma{}

	coinAddr := fakeCtx.CreateContract(coin.Contract)
	fakeCtx.RegisterContract("coin", coinAddr, coinAddr)
	coinContract := &coin.Coin{}

	coinCtx := fakeCtx.WithAddress(coinAddr)
	require.NoError(t, coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coinInit))

	require.NoError(t, coinContract.Approve(
		contractpb.WrapPluginContext(coinCtx.WithSender(addr1)),
		&coin.ApproveRequest{
			Spender: karmaAddr.MarshalPB(),
			Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
		}),
	)

	karmaCtx := fakeCtx.WithAddress(karmaAddr)
	require.NoError(t, karmaContract.Init(contractpb.WrapPluginContext(karmaCtx), &karmaInit))

	require.NoError(t,
		karmaContract.DepositCoin(
			contractpb.WrapPluginContext(karmaCtx.WithSender(addr1)),
			&ktypes.KarmaUserAmount{
				User:   user1,
				Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(20)},
			}),
	)

	require.NoError(t, karmaContract.AddKarma(
		contractpb.WrapPluginContext(karmaCtx.WithSender(addr1)),
		&ktypes.AddKarmaRequest{
			User:         user1,
			KarmaSources: awardsSoures,
		}),
	)

	_, err := karmaContract.GetSources(contractpb.WrapPluginContext(karmaCtx), &ktypes.GetSourceRequest{})
	require.NoError(t, err)

	// Deploy some contract on mock chain
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err := karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(1), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(21), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(3), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(17), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(11), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(20), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))

	contract1 := plugin.CreateAddress(addr1, 1)
	require.NoError(t, karma.AddOwnedContract(contractpb.WrapPluginContext(karmaCtx), addr1, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract1))

	contract2 := plugin.CreateAddress(addr1, 2)
	require.NoError(t, karma.AddOwnedContract(contractpb.WrapPluginContext(karmaCtx), addr1, contract2))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract2))

	block.Height = period + 1
	karmaCtx = karmaCtx.WithBlock(block)
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err = karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(3601), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(1), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(3), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(11), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(17), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(20), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract2))

	block.Height = 2*period + 1
	karmaCtx = karmaCtx.WithBlock(block)
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err = karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(7201), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(1), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(11), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(20), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract2))

	block.Height = 3*period + 1
	karmaCtx = karmaCtx.WithBlock(block)
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err = karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(10801), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(12), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract2))

	block.Height = 4*period + 1
	karmaCtx = karmaCtx.WithBlock(block)
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err = karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(14401), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(2), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))
	require.False(t, isActive(t, karmaContract, karmaCtx, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx, contract2))

	block.Height = 5*period + 1
	karmaCtx = karmaCtx.WithBlock(block)
	require.NoError(t, karma.Upkeep(contractpb.WrapPluginContext(karmaCtx)))
	upkeepState, err = karma.GetUpkeepState(contractpb.WrapPluginStaticContext(karmaCtx))
	require.NoError(t, err)
	require.Equal(t, uint64(18001), upkeepState.LastUpkeepHeight)
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award1"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award2"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award3"))
	require.Equal(t, int64(0), GetKarma(t, karmaCtx, addr1, "award4"))
	require.Equal(t, int64(2), GetKarma(t, karmaCtx, addr1, karma.CoinDeployToken))
	require.False(t, isActive(t, karmaContract, karmaCtx, contract1))
	require.False(t, isActive(t, karmaContract, karmaCtx, contract2))
}

/*
func TestKarmaCoinUpkeep(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Users: users,
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   10,
			Period: period,
		},
	}
	coinInit := coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user1, Balance: uint64(200)},
			{Owner: user2, Balance: uint64(200)},
		},
	}
	state, reg, pluginVm := karma.MockStateWithKarmaAndCoinT(t, &karmaInit, &coinInit)

	// Transfer karma to user
	karmaAddr, err := reg.Resolve("karma")
	require.NoError(t, err)
	karmaContract := &karma.Karma{}

	coinAddr, err := reg.Resolve("coin")
	require.NoError(t, err)
	coinContract := &coin.Coin{}

	coinCtx1 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr1, coinAddr, pluginVm),
	)
	require.NoError(t, coinContract.Approve(coinCtx1, &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
	}))

	karmaCtx1 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr1, karmaAddr, pluginVm),
	)
	require.NoError(t, karmaContract.DepositCoin(karmaCtx1, &ktypes.KarmaUserAmount{User: user1, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(104)}}))

	coinCtx2 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr2, coinAddr, pluginVm),
	)
	require.NoError(t, coinContract.Approve(coinCtx2, &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
	}))

	karmaCtx2 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr2, karmaAddr, pluginVm),
	)
	require.NoError(t, karmaContract.DepositCoin(karmaCtx2, &ktypes.KarmaUserAmount{User: user2, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(104)}}))

	// Deploy some contracts on mock chain
	kh := NewKarmaHandler(factory.RegistryV2, true, true)
	require.NoError(t, kh.Upkeep(state))
	require.Equal(t, uint64(1), binary.BigEndian.Uint64(state.Get(lastKarmaUpkeepKey)))

	require.Equal(t, int64(104), GetKarma(t, state, addr1, karma.CoinDeployToken))

	contract1 := karma.MockDeployEvmContract(t, state, addr1, 1, reg)
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract1))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))
	require.Equal(t, uint64(3601), binary.BigEndian.Uint64(state3600.Get(lastKarmaUpkeepKey)))

	require.Equal(t, int64(94), GetKarma(t, state3600, addr1, karma.CoinDeployToken))

	contract2 := karma.MockDeployEvmContract(t, state3600, addr1, 2, reg)
	contract3 := karma.MockDeployEvmContract(t, state3600, addr1, 3, reg)
	contract4 := karma.MockDeployEvmContract(t, state3600, addr1, 4, reg)
	contractAddr2 := karma.MockDeployEvmContract(t, state3600, addr2, 1, reg)

	state7200 := common.MockStateAt(state, 2*period+1)
	require.NoError(t, kh.Upkeep(state7200))
	require.Equal(t, uint64(7201), binary.BigEndian.Uint64(state7200.Get(lastKarmaUpkeepKey)))

	require.Equal(t, int64(54), GetKarma(t, state7200, addr1, karma.CoinDeployToken))
	require.Equal(t, int64(94), GetKarma(t, state7200, addr2, karma.CoinDeployToken))

	contract5 := karma.MockDeployEvmContract(t, state7200, addr1, 5, reg)
	contract6 := karma.MockDeployEvmContract(t, state7200, addr1, 6, reg)

	require.True(t, isActive(t, karmaContract, karmaCtx1, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract2))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract3))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract4))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract5))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract6))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contractAddr2))

	state10800 := common.MockStateAt(state, 3*period+1)
	require.NoError(t, kh.Upkeep(state10800))
	require.Equal(t, uint64(10801), binary.BigEndian.Uint64(state10800.Get(lastKarmaUpkeepKey)))

	require.Equal(t, int64(4), GetKarma(t, state10800, addr1, karma.CoinDeployToken))
	require.Equal(t, int64(84), GetKarma(t, state10800, addr2, karma.CoinDeployToken))

	require.False(t, isActive(t, karmaContract, karmaCtx1, contract1))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract2))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract3))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract4))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract5))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contract6))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contractAddr2))

	state14400 := common.MockStateAt(state, 4*period+1)
	require.NoError(t, kh.Upkeep(state14400))
	require.Equal(t, uint64(14401), binary.BigEndian.Uint64(state14400.Get(lastKarmaUpkeepKey)))

	require.Equal(t, int64(4), GetKarma(t, state14400, addr1, karma.CoinDeployToken))
	require.Equal(t, int64(74), GetKarma(t, state14400, addr2, karma.CoinDeployToken))

	require.False(t, isActive(t, karmaContract, karmaCtx1, contract1))
	require.False(t, isActive(t, karmaContract, karmaCtx1, contract2))
	require.False(t, isActive(t, karmaContract, karmaCtx1, contract3))
	require.False(t, isActive(t, karmaContract, karmaCtx1, contract4))
	require.False(t, isActive(t, karmaContract, karmaCtx1, contract5))
	require.False(t, isActive(t, karmaContract, karmaCtx1, contract6))
	require.True(t, isActive(t, karmaContract, karmaCtx1, contractAddr2))
}
*/
func GetKarma(t *testing.T, karmaContractCtx lplugin.StaticContext, user loom.Address, sourceName string) int64 {
	userState, err := karma.GetUserState(contractpb.WrapPluginStaticContext(karmaContractCtx), user)
	require.NoError(t, err)

	for _, source := range userState.SourceStates {
		if source.Name == sourceName {
			return source.Count.Value.Int64()
		}
	}
	return 0
}

func isActive(t *testing.T, karmaContract *karma.Karma, ctx lplugin.StaticContext, contract loom.Address) bool {
	active, err := karma.IsContractActive(contractpb.WrapPluginStaticContext(ctx), contract)
	require.NoError(t, err)
	return active
}
