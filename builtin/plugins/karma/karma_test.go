package karma

import (
	"github.com/loomnetwork/go-loom"
	"testing"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin"
	ktypes "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr4 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	maxKarma int64	= 10000
	oraclePublicAddress = addr1.Local.String()
	oraclePublicAddress2= addr2.Local.String()
	oraclePublicAddress3= addr3.Local.String()
	userPublicAddress 	= addr4.Local.String()

	sources 			= map[string]int64{
								"sms": 10.0,
								"oauth": 10.0,
								"token": 5.0,
							}
	sourceStates 		= map[string]int64{
								"sms": 1.0,
								"oauth": 5.0,
								"token": 10.0,
							}
	extremeSourceStates = map[string]int64{
								"sms": 1000.0,
								"oauth": 5000.0,
								"token": 10.0,
							}
	deleteSourceKeys	= []string{"sms", "oauth"}
)



func TestKarma_Init(t *testing.T) {

	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			OraclePublicAddress: oraclePublicAddress,
			Sources: sources,
		},
	})
	require.Nil(t, err)

	config, err := contract.GetConfig(ctx, &ktypes.KarmaUser{})
	require.Nil(t, err)
	require.Equal(t, maxKarma, config.MaxKarma)
	require.Equal(t, oraclePublicAddress, config.Oracle.Address)
	require.Equal(t, sources, config.Sources)

}

func TestKarma_validateOracle(t *testing.T) {

	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			OraclePublicAddress: oraclePublicAddress,
			Sources: sources,
		},
	})
	require.Nil(t, err)

	err = contract.validateOracle(ctx, &ktypes.KarmaUser{
		Address:oraclePublicAddress,
	})
	require.Nil(t, err)

	err = contract.validateOracle(ctx, &ktypes.KarmaUser{
		Address:userPublicAddress,
	})
	require.NotNil(t, err)

}

func TestKarma_LifeCycleTest(t *testing.T) {

	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Karma{}

	//Init Test
	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			OraclePublicAddress: oraclePublicAddress,
			Sources: sources,
		},
	})
	require.Nil(t, err)

	//GetState Test
	ko := &ktypes.KarmaUser{
		Address:userPublicAddress,
	}

	oracle := &ktypes.KarmaUser{
		Address:oraclePublicAddress,
	}

	err = contract.AddNewSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:ko,
		Oracle:oracle,
		SourceStates: sourceStates,

	})
	require.Nil(t, err)

	//GetState Test
	state, err := contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, sourceStates, state.SourceStates)

	//GetTotal Test
	karmaTotal, err := contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(110), karmaTotal.Count)


	//UpdateSourcesForUser Test
	err = contract.UpdateSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:ko,
		Oracle:oracle,
		SourceStates: extremeSourceStates,

	})
	require.Nil(t, err)

	//GetState after UpdateSourcesForUser Test to test the change
	state, err = contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, extremeSourceStates, state.SourceStates)

	//GetState after UpdateSourcesForUser and also MaxKarma Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(10000), karmaTotal.Count)

	//DeleteSourcesForUser Test
	err = contract.DeleteSourcesForUser(ctx, &ktypes.KarmaStateKeyUser{
		User:ko,
		Oracle:oracle,
		StateKeys: deleteSourceKeys,

	})
	require.Nil(t, err)

	//GetState after DeleteSourcesForUser Test
	state, err = contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, map[string]int64{"token": 10}, state.SourceStates)

	//GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(50), karmaTotal.Count)


	//Update entire config anf change oracle
	err = contract.UpdateConfig(ctx, &ktypes.KarmaParamsOracle{
		Params: &ktypes.KarmaParams{
			MaxKarma: maxKarma,
			OraclePublicAddress: oraclePublicAddress2,
			Sources: sources,
		},
		Oracle:oracle,
	})
	require.Nil(t, err)

	//Testing old oracle to be disabled
	err = contract.UpdateSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:ko,
		Oracle:oracle,
		SourceStates: sourceStates,
	})
	require.NotNil(t, err)

	//Testing new oracle
	oracle2 := &ktypes.KarmaUser{
		Address:oraclePublicAddress2,
	}
	err = contract.UpdateSourcesForUser(ctx, &ktypes.KarmaStateUser{
		User:ko,
		Oracle:oracle2,
		SourceStates: sourceStates,
	})
	require.Nil(t, err)

	state, err = contract.GetState(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, sourceStates, state.SourceStates)

	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(110), karmaTotal.Count)


	//Update config in parts for max karma
	err = contract.UpdateConfigMaxKarma(ctx, &ktypes.KarmaParamsOracleNewMaxKarma{
		MaxKarma:10,
		Oracle:oracle2,
	})
	require.Nil(t, err)

	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(10), karmaTotal.Count)

	//Update config in parts for new oracle
	oracle3 := &ktypes.KarmaUser{
		Address:oraclePublicAddress3,
	}
	err = contract.UpdateConfigOracle(ctx, &ktypes.KarmaParamsOracleNewOracle{
		NewOraclePublicAddress:oraclePublicAddress3,
		Oracle:oracle2,
	})
	require.Nil(t, err)

	err = contract.UpdateConfigMaxKarma(ctx, &ktypes.KarmaParamsOracleNewMaxKarma{
		MaxKarma:12,
		Oracle:oracle3,
	})
	require.Nil(t, err)

	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(12), karmaTotal.Count)
}

