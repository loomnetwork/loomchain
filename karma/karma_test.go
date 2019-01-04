package karma

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/receipts/common"
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

	deploySource = []*ktypes.KarmaSourceReward{
		{Name: karma.DeployToken, Reward:3, Target:ktypes.KarmaSourceTarget_DEPLOY},
	}

	users = []*ktypes.KarmaAddressSource{
		{ User: user1, Sources: []*ktypes.KarmaSource{{ Name: karma.DeployToken, Count: 104}}},
		{ User: user2, Sources: []*ktypes.KarmaSource{{ Name:karma.DeployToken, Count: 104}}},
	}
)

func TestKarma(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParams{
			Cost:   10,
			Source: karma.DeployToken,
			Period: period,
		},
	}
	state := karma.MockStateWithKarma(t, karmaInit)

	kh := NewKarmaHandler(factory.RegistryV2, true)
	require.NoError(t, kh.Upkeep(state))
	require.Equal(t, int64(104), karma.GetKarma(t, state, *user1, karma.DeployToken))

	contract1 := karma.MockDeployEvmContract(t, state, addr1, 1)
	require.True(t, karma.IsActive(t, state, contract1))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))
	require.Equal(t, int64(94), karma.GetKarma(t, state3600, *user1, karma.DeployToken))

	contract2 := karma.MockDeployEvmContract(t, state3600, addr1, 2)
	contract3 := karma.MockDeployEvmContract(t, state3600, addr1, 3)
	contract4 := karma.MockDeployEvmContract(t, state3600, addr1, 4)
	contractAddr2 := karma.MockDeployEvmContract(t, state3600, addr2, 1)

	state7200 := common.MockStateAt(state, 2*period+1)
	require.NoError(t, kh.Upkeep(state7200))
	require.Equal(t, int64(54), karma.GetKarma(t, state7200, *user1, karma.DeployToken))
	require.Equal(t, int64(94), karma.GetKarma(t, state7200, *user2, karma.DeployToken))

	contract5 := karma.MockDeployEvmContract(t, state7200, addr1, 5)
	contract6 := karma.MockDeployEvmContract(t, state7200, addr1, 6)

	require.True(t, karma.IsActive(t, state7200, contract1))
	require.True(t, karma.IsActive(t, state7200, contract2))
	require.True(t, karma.IsActive(t, state7200, contract3))
	require.True(t, karma.IsActive(t, state7200, contract4))
	require.True(t, karma.IsActive(t, state7200, contract5))
	require.True(t, karma.IsActive(t, state7200, contract6))
	require.True(t, karma.IsActive(t, state7200, contractAddr2))

	state10800 := common.MockStateAt(state, 3*period+1)
	require.NoError(t, kh.Upkeep(state10800))
	require.Equal(t, int64(4), karma.GetKarma(t, state10800, *user1, karma.DeployToken))
	require.Equal(t, int64(84), karma.GetKarma(t, state10800, *user2, karma.DeployToken))

	require.True(t, karma.IsActive(t, state10800, contract1))
	require.True(t, karma.IsActive(t, state10800, contract2))
	require.True(t, karma.IsActive(t, state10800, contract3))
	require.True(t, karma.IsActive(t, state10800, contract4))
	require.True(t, karma.IsActive(t, state10800, contract5))
	require.False(t, karma.IsActive(t, state10800, contract6))
	require.True(t, karma.IsActive(t, state10800, contractAddr2))


	state14400 := common.MockStateAt(state, 4*period+1)
	require.NoError(t, kh.Upkeep(state14400))
	require.Equal(t, int64(4), karma.GetKarma(t, state14400, *user1, karma.DeployToken))
	require.Equal(t, int64(74), karma.GetKarma(t, state14400, *user2, karma.DeployToken))

	require.False(t, karma.IsActive(t, state14400, contract1))
	require.False(t, karma.IsActive(t, state14400, contract2))
	require.False(t, karma.IsActive(t, state14400, contract3))
	require.False(t, karma.IsActive(t, state14400, contract4))
	require.False(t, karma.IsActive(t, state14400, contract5))
	require.False(t, karma.IsActive(t, state14400, contract6))
	require.True(t, karma.IsActive(t, state14400, contractAddr2))
}
