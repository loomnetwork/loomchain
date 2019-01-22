package karma

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
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

	user1   = types_addr1
	user2   = types_addr2

	emptySourceStates = []*ktypes.KarmaSource{}

	users = []*ktypes.KarmaAddressSource{
		{ User: user1, Sources: emptySourceStates },
		{ User: user2, Sources: emptySourceStates },
	}
)

func TestKarmaCoinUpkeep(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParams{
			Cost:   10,
			Period: period,
		},
	}
	coinInit := coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{ Owner:   user1,	Balance: uint64(200) },
			{ Owner:   user2,	Balance: uint64(200) },
		},
	}
	state, reg, pluginVm := karma.MockStateWithKarmaAndCoin(t, &karmaInit, &coinInit, "mockAppDb")

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
	require.NoError(t,coinContract.Approve(coinCtx1, &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
	}))

	karmaCtx1 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr1, karmaAddr, pluginVm),
	)
	require.NoError(t, karmaContract.DepositCoin(karmaCtx1, &ktypes.KarmaUserAmount{ User: user1, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(104)}}))

	coinCtx2 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr2, coinAddr, pluginVm),
	)
	require.NoError(t,coinContract.Approve(coinCtx2, &coin.ApproveRequest{
		Spender: karmaAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
	}))

	karmaCtx2 := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr2, karmaAddr, pluginVm),
	)
	require.NoError(t, karmaContract.DepositCoin(karmaCtx2, &ktypes.KarmaUserAmount{ User: user2, Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(104)}}))

	// Deploy some contracts on mock chain
	kh := NewKarmaHandler(factory.RegistryV2, true)
	require.NoError(t, kh.Upkeep(state))
	require.Equal(t, int64(104), GetKarma(t, state, *user1, karma.CoinDeployToken, reg))

	contract1 := karma.MockDeployEvmContract(t, state, addr1, 1, reg)
	require.True(t, IsActive(t, state, contract1, reg))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))
	require.Equal(t, int64(94), GetKarma(t, state3600, *user1, karma.CoinDeployToken, reg))

	contract2 := karma.MockDeployEvmContract(t, state3600, addr1, 2, reg)
	contract3 := karma.MockDeployEvmContract(t, state3600, addr1, 3, reg)
	contract4 := karma.MockDeployEvmContract(t, state3600, addr1, 4, reg)
	contractAddr2 := karma.MockDeployEvmContract(t, state3600, addr2, 1, reg)

	state7200 := common.MockStateAt(state, 2*period+1)
	require.NoError(t, kh.Upkeep(state7200))
	require.Equal(t, int64(54), GetKarma(t, state7200, *user1, karma.CoinDeployToken, reg))
	require.Equal(t, int64(94), GetKarma(t, state7200, *user2, karma.CoinDeployToken, reg))

	contract5 := karma.MockDeployEvmContract(t, state7200, addr1, 5, reg)
	contract6 := karma.MockDeployEvmContract(t, state7200, addr1, 6, reg)

	require.True(t, IsActive(t, state7200, contract1, reg))
	require.True(t, IsActive(t, state7200, contract2, reg))
	require.True(t, IsActive(t, state7200, contract3, reg))
	require.True(t, IsActive(t, state7200, contract4, reg))
	require.True(t, IsActive(t, state7200, contract5, reg))
	require.True(t, IsActive(t, state7200, contract6, reg))
	require.True(t, IsActive(t, state7200, contractAddr2, reg))

	state10800 := common.MockStateAt(state, 3*period+1)
	require.NoError(t, kh.Upkeep(state10800))
	require.Equal(t, int64(4), GetKarma(t, state10800, *user1, karma.CoinDeployToken, reg))
	require.Equal(t, int64(84), GetKarma(t, state10800, *user2, karma.CoinDeployToken, reg))

	require.False(t, IsActive(t, state10800, contract1, reg))
	require.True(t, IsActive(t, state10800, contract2, reg))
	require.True(t, IsActive(t, state10800, contract3, reg))
	require.True(t, IsActive(t, state10800, contract4, reg))
	require.True(t, IsActive(t, state10800, contract5, reg))
	require.True(t, IsActive(t, state10800, contract6, reg))
	require.True(t, IsActive(t, state10800, contractAddr2, reg))


	state14400 := common.MockStateAt(state, 4*period+1)
	require.NoError(t, kh.Upkeep(state14400))
	require.Equal(t, int64(4), GetKarma(t, state14400, *user1, karma.CoinDeployToken, reg))
	require.Equal(t, int64(74), GetKarma(t, state14400, *user2, karma.CoinDeployToken, reg))

	require.False(t, IsActive(t, state14400, contract1, reg))
	require.False(t, IsActive(t, state14400, contract2, reg))
	require.False(t, IsActive(t, state14400, contract3, reg))
	require.False(t, IsActive(t, state14400, contract4, reg))
	require.False(t, IsActive(t, state14400, contract5, reg))
	require.False(t, IsActive(t, state14400, contract6, reg))
	require.True(t, IsActive(t, state14400, contractAddr2, reg))
}



func IsActive(t *testing.T, state loomchain.State, contract loom.Address, reg registry.Registry) bool {
	return karma.GetKarmaState(t, state, reg).Has(karma.ContractActiveRecordKey(contract))
}

func GetKarma(t *testing.T, state loomchain.State, user types.Address, sourceName string, reg registry.Registry) int64 {
	karmaState := karma.GetKarmaState(t, state, reg)

	userStateKey := karma.UserStateKey(&user)
	data := karmaState.Get(userStateKey)
	var userState ktypes.KarmaState
	require.NoError(t, proto.Unmarshal(data, &userState))

	for _, source := range userState.SourceStates {
		if source.Name == sourceName {
			return source.Count.Value.Int64()
		}
	}
	return 0
}