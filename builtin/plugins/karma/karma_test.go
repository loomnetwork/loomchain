package karma

import (
	"github.com/loomnetwork/go-loom"
	"testing"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/stretchr/testify/require"
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

	maxKarma int64	= 10000
	oracle 	= types_addr1
	oracle2	= types_addr2
	oracle3	= types_addr3
	user	= types_addr4
	

	sources = []*SourceReward{
		&SourceReward{"sms", 1},
		&SourceReward{"oauth", 3},
		&SourceReward{"token", 4},
	}

	sourceStates = []*Source{
		&Source{"sms", 1},
		&Source{"oauth", 5},
		&Source{"token", 10},
	}

	extremeSourceStates = []*Source{
		&Source{"sms", 1000},
		&Source{"oauth", 5000},
		&Source{"token", 10},
	}
	
	users   = []*AddressSource{
		{user, sourceStates},
		{oracle, sourceStates},
	}
	deleteSourceKeys	= []string{"sms", "oauth"}
)



func TestKarmaInit(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			Oracle: oracle,
			Sources: sources,
			Users: users,
		},
	})
	require.Nil(t, err)

	config, err := contract.GetConfig(ctx, oracle)
	require.Nil(t, err)
	require.Equal(t, maxKarma, config.MaxKarma)
	require.Equal(t, oracle, config.Oracle)
	require.Equal(t, sources, config.Sources)
	for _, u := range users {
		require.True(t, ctx.Has(GetUserStateKey(u.User)))
		state, err := contract.GetState(ctx, u.User)
		require.NoError(t, err)
		require.Equal(t, len(sourceStates), len(state.SourceStates))
	}
}

func TestKarmavalidateOracle(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			Oracle: oracle,
			Sources: sources,
		},
	})
	require.Nil(t, err)

	err = contract.validateOracle(ctx, oracle)
	require.Nil(t, err)

	err = contract.validateOracle(ctx, user)
	require.NotNil(t, err)

}

func TestKarmaLifeCycleTest(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}

	//Init Test
	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma:   maxKarma,
			Oracle:     oracle,
			Sources:    sources,
		},
	})
	require.Nil(t, err)

	//GetState Test
	ko := user

	//UpdateSourcesForUser Test
	err = contract.UpdateSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:         ko,
		Oracle:       oracle,
		SourceStates: extremeSourceStates,
	})
	require.Nil(t, err)

	//GetState after UpdateSourcesForUser Test to test the change
	state, err := contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, extremeSourceStates, state.SourceStates)

	//GetState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err := contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(10000), karmaTotal.Count)

	//DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:      ko,
		Oracle:    oracle,
		StateKeys: deleteSourceKeys,
	})
	require.Nil(t, err)

	//GetState after DeleteSourcesForUser Test
	state, err = contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, []*Source{&Source{"token", 10}}, state.SourceStates)

	//GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(40), karmaTotal.Count)

	//Update entire config anf change oracle
	err = contract.UpdateConfig(ctx, &ktypes.KarmaParamsValidator{
		Params: &ktypes.KarmaParams{
			MaxKarma:   maxKarma,
			Oracle:     oracle2,
			Sources:    sources,
		},
		Oracle:    oracle,
	})
	require.NotNil(t, err)

}

