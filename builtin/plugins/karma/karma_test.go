package karma

import (
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr4 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr5 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c6")

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()
	types_addr3 = addr3.MarshalPB()
	types_addr4 = addr4.MarshalPB()
	types_addr5 = addr5.MarshalPB()

	oracle  = types_addr1
	oracle2 = types_addr2
	oracle3 = types_addr3
	user    = types_addr4

	sources = []*ktypes.KarmaSourceReward{
		{"sms", 1},
		{"oauth", 3},
		{"token", 4},
	}

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
		state, err := contract.GetState(ctx, u.User)
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

	//UpdateSourcesForUser Test
	err = contract.UpdateSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:         ko,
		Oracle:       oracle,
		SourceStates: extremeSourceStates,
	})
	require.NoError(t, err)

	//GetState after UpdateSourcesForUser Test to test the change
	state, err := contract.GetState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, extremeSourceStates, state.SourceStates)

	//GetState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetTotal(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, int64(16040), karmaTotal.Count)

	//DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:      ko,
		Oracle:    oracle,
		StateKeys: deleteSourceKeys,
	})
	require.NoError(t, err)

	//GetState after DeleteSourcesForUser Test
	state, err = contract.GetState(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, []*ktypes.KarmaSource{{"token", 10}}, state.SourceStates)

	//GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.NoError(t, err)
	require.Equal(t, int64(40), karmaTotal.Count)

	isOracle, err := contract.IsOracle(ctx, oracle)
	require.True(t, isOracle)

	isOracle, err = contract.IsOracle(ctx, oracle2)
	require.False(t, isOracle)

	//Update entire config anf change oracle
	err = contract.UpdateOracle(ctx, &ktypes.KarmaNewOracleValidator{
		OldOracle: oracle,
		NewOracle: oracle2,
	})
	require.NoError(t, err)

	isOracle, err = contract.IsOracle(ctx, oracle)
	require.False(t, isOracle)

	isOracle, err = contract.IsOracle(ctx, oracle2)
	require.True(t, isOracle)
}
