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

	sources 			= map[string]int64{
								"sms": 10,
								"oauth": 10,
								"token": 5,
							}
	sourceStates 		= map[string]int64{
								"sms": 1,
								"oauth": 5,
								"token": 10,
							}
	extremeSourceStates = map[string]int64{
								"sms": 1000,
								"oauth": 5000,
								"token": 10,
							}
	deleteSourceKeys	= []string{"sms", "oauth"}
)



func TestKarmaInit(t *testing.T) {
	validator := &loom.Validator{
		PubKey: []byte(addr5.String()),
		Power: 2,
	}

	var validators []*loom.Validator
	validators = append(validators, validator)
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			Oracle: oracle,
			Sources: sources,
			Validators: validators,
		},
	})
	require.Nil(t, err)

	config, err := contract.GetConfig(ctx, oracle)
	require.Nil(t, err)
	require.Equal(t, maxKarma, config.MaxKarma)
	require.Equal(t, oracle, config.Oracle)
	require.Equal(t, sources, config.Sources)

}

func TestKarmavalidateOracle(t *testing.T) {

	validator := &loom.Validator{
		PubKey: []byte(addr5.String()),
		Power: 2,
	}

	var validators []*loom.Validator
	validators = append(validators, validator)
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &Karma{}

	err := contract.Init(ctx, &ktypes.KarmaInitRequest{
		Params: &Params{
			MaxKarma: maxKarma,
			Oracle: oracle,
			Sources: sources,
			Validators: validators,
		},
	})
	require.Nil(t, err)

	err = contract.validateOracle(ctx, oracle)
	require.Nil(t, err)

	err = contract.validateOracle(ctx, user)
	require.NotNil(t, err)

}

func TestKarmaLifeCycleTest(t *testing.T) {

	validator := &loom.Validator{
		PubKey: []byte(addr5.String()),
		Power:  2,
	}

	var validators []*loom.Validator
	validators = append(validators, validator)

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
			Validators: validators,
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
	require.Equal(t, map[string]int64{"token": 10}, state.SourceStates)

	//GetTotal after DeleteSourcesForUser Test to test the change
	karmaTotal, err = contract.GetTotal(ctx, ko)
	require.Nil(t, err)
	require.Equal(t, int64(50), karmaTotal.Count)

	//Update entire config anf change oracle
	err = contract.UpdateConfig(ctx, &ktypes.KarmaParamsValidator{
		Params: &ktypes.KarmaParams{
			MaxKarma:   maxKarma,
			Oracle:     oracle2,
			Sources:    sources,
			Validators: nil,
		},
		Validator: validator,
		Oracle:    oracle,
	})
	require.NotNil(t, err)

}

