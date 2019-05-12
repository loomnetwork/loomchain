package chainconfig

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	plugintypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/stretchr/testify/suite"
)

var (
	pubKey1                                            = "1V7jqasQYZIdHJtrjD9Raq4rOALsAL1a0yQytoQp46g="
	pubKey2                                            = "JHFJjkpXUJLuTTl+kOJ3I6EA1TnKtIOUxo7uPGlcPTQ="
	pubKey3                                            = "l/xG3rd63kAzflA2hMQgKq3CDDuKzseXIzAc/MS8FPI="
	pubKey4                                            = "umC8MrxDsffG9153juF61840dDCEIrhKVxyI72UkoSw="
	pubKeyB64_1, pubKeyB64_2, pubKeyB64_3, pubKeyB64_4 []byte
)

type ChainConfigTestSuite struct {
	suite.Suite
}

func (c *ChainConfigTestSuite) SetupTest() {
}

func TestChainConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ChainConfigTestSuite))
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}

func (c *ChainConfigTestSuite) TestFeatureFlagEnabledSingleValidator() {
	require := c.Require()
	featureName := "hardfork"
	featureName2 := "test-ft"
	featureName3 := "test2-ft"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ := encoder.DecodeString(pubKey1)
	chainID := "default"
	addr1 := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	//setup fake contract
	validators := []*loom.Validator{
		&loom.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
	}
	pctx := plugin.CreateFakeContext(addr1, addr1).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    time.Now().Unix(),
	}).WithValidators(validators)
	//Init fake coin contract
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err := coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{},
	})
	require.NoError(err)

	//Init fake dposv2 contract
	dposv2Contract := dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(dposv2.Contract)
	pctx = pctx.WithAddress(dposv2Addr)
	ctx := contractpb.WrapPluginContext(pctx)

	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	//setup chainconfig contract
	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Params: &Params{
			VoteThreshold:         66,
			NumBlockConfirmations: 10,
		},
		Features: []*Feature{
			&Feature{
				Name:       featureName2,
				Status:     FeaturePending,
				Percentage: 0,
			},
			&Feature{
				Name:       featureName3,
				Status:     FeatureWaiting,
				Percentage: 100,
			},
		},
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(ctx, &AddFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	getFeature, err := chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(0), getFeature.Feature.Percentage)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName2,
	})
	require.NoError(err)
	require.Equal(featureName2, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(0), getFeature.Feature.Percentage)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName3,
	})
	require.NoError(err)
	require.Equal(featureName3, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_WAITING, getFeature.Feature.Status)
	require.Equal(uint64(100), getFeature.Feature.Percentage)

	listFeatures, err := chainconfigContract.ListFeatures(ctx, &ListFeaturesRequest{})
	require.NoError(err)
	require.Equal(3, len(listFeatures.Features))

	err = chainconfigContract.EnableFeature(ctx, &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(100), getFeature.Feature.Percentage)

	err = chainconfigContract.SetParams(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &SetParamsRequest{
		Params: &Params{
			VoteThreshold:         60,
			NumBlockConfirmations: 5,
		},
	})
	require.NoError(err)

	getParams, err := chainconfigContract.GetParams(ctx, &GetParamsRequest{})
	require.NoError(err)
	require.Equal(uint64(60), getParams.Params.VoteThreshold)
	require.Equal(uint64(5), getParams.Params.NumBlockConfirmations)

	featureEnabled, err := chainconfigContract.FeatureEnabled(ctx, &plugintypes.FeatureEnabledRequest{
		Name:       featureName,
		DefaultVal: true,
	})
	require.Equal(true, featureEnabled.Value)
}

func (c *ChainConfigTestSuite) TestPermission() {
	require := c.Require()
	featureName := "hardfork"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ := encoder.DecodeString(pubKey1)
	pubKeyB64_2, _ := encoder.DecodeString(pubKey2)
	addr1 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	addr2 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_2)}
	//setup fake contract
	validators := []*loom.Validator{
		&loom.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
	}
	pctx := plugin.CreateFakeContext(addr1, addr1).WithValidators(validators)
	//Init fake coin contract
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err := coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{},
	})
	require.NoError(err)

	//Init fake dposv2 contract
	dposv2Contract := dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(dposv2.Contract)
	pctx = pctx.WithAddress(dposv2Addr)
	ctx := contractpb.WrapPluginContext(pctx)

	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	//setup chainconfig contract
	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Params: &Params{
			VoteThreshold:         66,
			NumBlockConfirmations: 10,
		},
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(ctx, &AddFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &AddFeatureRequest{
		Names: []string{"newFeature"},
	})
	require.Equal(ErrNotAuthorized, err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.Equal(ErrNotAuthorized, err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	err = chainconfigContract.SetParams(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &SetParamsRequest{
		Params: &Params{
			VoteThreshold:         60,
			NumBlockConfirmations: 5,
		},
	})
	require.NoError(err)

	err = chainconfigContract.SetParams(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &SetParamsRequest{
		Params: &Params{
			VoteThreshold:         60,
			NumBlockConfirmations: 5,
		},
	})
	require.Equal(ErrNotAuthorized, err)
}

func (c *ChainConfigTestSuite) TestFeatureFlagEnabledFourValidators() {
	require := c.Require()
	featureName := "hardfork"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ = encoder.DecodeString(pubKey1)
	addr1 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	pubKeyB64_2, _ = encoder.DecodeString(pubKey2)
	addr2 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_2)}
	pubKeyB64_3, _ = encoder.DecodeString(pubKey3)
	addr3 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_3)}
	pubKeyB64_4, _ = encoder.DecodeString(pubKey4)
	addr4 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_4)}

	pctx := plugin.CreateFakeContext(addr1, addr1)
	validators := []*loom.Validator{
		&loom.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_2,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_3,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_4,
			Power:  10,
		},
	}
	pctx = pctx.WithValidators(validators)

	//Init fake coin contract
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err := coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{},
	})
	require.NoError(err)

	//Init fake dposv2 contract
	dposv2Contract := dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(dposv2.Contract)
	pctx = pctx.WithAddress(dposv2Addr)
	ctx := contractpb.WrapPluginContext(pctx)

	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Params: &Params{
			VoteThreshold:         66,
			NumBlockConfirmations: 10,
		},
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &AddFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr4)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.Error(ErrFeatureAlreadyEnabled)

	getFeature, err := chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(50), getFeature.Feature.Percentage)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	getFeature, err = chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(75), getFeature.Feature.Percentage)

	buildNumber := uint64(1000)
	enabledFeatures, err := EnableFeatures(ctx, 20, buildNumber)
	require.NoError(err)
	require.Equal(0, len(enabledFeatures))

	getFeature, err = chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_WAITING, getFeature.Feature.Status)
	require.Equal(uint64(75), getFeature.Feature.Percentage)

	enabledFeatures, err = EnableFeatures(ctx, 31, buildNumber)
	require.NoError(err)
	require.Equal(1, len(enabledFeatures))

	getFeature, err = chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_ENABLED, getFeature.Feature.Status)
	require.Equal(uint64(75), getFeature.Feature.Percentage)

	chainconfigContract.FeatureEnabled(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &plugintypes.FeatureEnabledRequest{
		Name: featureName,
	})
}

func (c *ChainConfigTestSuite) TestUnsupportedFeatureEnabled() {
	require := c.Require()
	featureName := "hardfork"
	featureName2 := "test-ft"
	featureName3 := "test2-ft"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ := encoder.DecodeString(pubKey1)
	chainID := "default"
	addr1 := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	//setup dposv2 fake contract
	pctx := plugin.CreateFakeContext(addr1, addr1).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    time.Now().Unix(),
	})

	//Init fake coin contract
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err := coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{},
	})
	require.NoError(err)

	//Init fake dposv2 contract
	dposv2Contract := dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(dposv2.Contract)
	pctx = pctx.WithAddress(dposv2Addr)
	ctx := contractpb.WrapPluginContext(pctx)
	validators := []*dposv2.Validator{
		&dposv2.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
	}
	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	//setup chainconfig contract
	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Params: &Params{
			VoteThreshold:         66,
			NumBlockConfirmations: 10,
		},
		Features: []*Feature{
			&Feature{
				Name:       featureName2,
				Status:     FeaturePending,
				Percentage: 0,
			},
			&Feature{
				Name:       featureName3,
				Status:     FeatureWaiting,
				Percentage: 100,
			},
		},
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(ctx, &AddFeatureRequest{
		Names:       []string{featureName},
		BuildNumber: 1000,
	})
	require.NoError(err)

	getFeature, err := chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(0), getFeature.Feature.Percentage)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName2,
	})
	require.NoError(err)
	require.Equal(featureName2, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.Feature.Status)
	require.Equal(uint64(0), getFeature.Feature.Percentage)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName3,
	})
	require.NoError(err)
	require.Equal(featureName3, getFeature.Feature.Name)
	require.Equal(cctypes.Feature_WAITING, getFeature.Feature.Status)
	require.Equal(uint64(100), getFeature.Feature.Percentage)

	listFeatures, err := chainconfigContract.ListFeatures(ctx, &ListFeaturesRequest{})
	require.NoError(err)
	require.Equal(3, len(listFeatures.Features))

	err = chainconfigContract.EnableFeature(ctx, &EnableFeatureRequest{
		Names: []string{featureName},
	})
	require.NoError(err)

	buildNumber := uint64(10)
	_, err = EnableFeatures(ctx, 100, buildNumber)
	_, err = EnableFeatures(ctx, 1000, buildNumber)
	require.Equal(ErrFeatureNotSupported, err)

	buildNumber = uint64(2000)
	_, err = EnableFeatures(ctx, 1000, buildNumber)
	require.NoError(err)
}

func (c *ChainConfigTestSuite) TestChainConfigFourValidators() {
	require := c.Require()
	configName := "dpos.feeFloor"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ = encoder.DecodeString(pubKey1)
	addr1 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	pubKeyB64_2, _ = encoder.DecodeString(pubKey2)
	addr2 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_2)}
	pubKeyB64_3, _ = encoder.DecodeString(pubKey3)
	addr3 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_3)}
	pubKeyB64_4, _ = encoder.DecodeString(pubKey4)
	addr4 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_4)}

	pctx := plugin.CreateFakeContext(addr1, addr1)
	validators := []*loom.Validator{
		&loom.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_2,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_3,
			Power:  10,
		},
		&loom.Validator{
			PubKey: pubKeyB64_4,
			Power:  10,
		},
	}
	pctx = pctx.WithValidators(validators)

	//Init fake coin contract
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err := coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{},
	})
	require.NoError(err)

	//Init fake dposv2 contract
	dposv2Contract := dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(dposv2.Contract)
	pctx = pctx.WithAddress(dposv2Addr)
	ctx := contractpb.WrapPluginContext(pctx)

	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Params: &Params{
			VoteThreshold:         66,
			NumBlockConfirmations: 10,
		},
	})
	require.NoError(err)

	setConfigRequest := &SetConfigRequest{
		Name:  configName,
		Value: "40",
	}

	err = chainconfigContract.AddConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &AddConfigRequest{
		Names:         []string{configName},
		VoteThreshold: 80,
		BuildNumber:   0,
	})
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr4)), setConfigRequest)
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr2)), setConfigRequest)
	require.NoError(err)

	setConfigRequest.Value = "30"
	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr3)), setConfigRequest)
	require.NoError(err)

	config, err := chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	require.NoError(err)
	require.Equal(cctypes.Config_VOTING, config.Config.Status)
	proposal := getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(50), proposal.Percentage)
	require.Equal("40", proposal.Value)

	setConfigRequest.Value = "40"
	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr3)), setConfigRequest)
	require.NoError(err)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	proposal = getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(75), proposal.Percentage)
	require.Equal("40", proposal.Value)

	configs, err := SetConfigs(contractpb.WrapPluginContext(pctx), 70, 0)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	require.Nil(config.Config.Settlement)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), setConfigRequest)
	require.NoError(err)

	configs, err = SetConfigs(contractpb.WrapPluginContext(pctx), 70, 0)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	proposal = getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(100), proposal.Percentage)
	require.NotNil(config.Config.Settlement)
	require.Equal(ConfigSettled, config.Config.Status)
	require.Equal("40", config.Config.Settlement.Value)

	configs, err = SetConfigs(contractpb.WrapPluginContext(pctx), 100, 0)
	require.Equal(1, len(configs))
	require.Equal(configName, configs[0].Name)
	require.Equal("40", configs[0].Settlement.Value)
	//storeStateKey := util.PrefixKey([]byte("config"), []byte(configName))
	pctx.SetConfig(configName, configs[0].Settlement.Value)

	cfg := pctx.ChainConfig()
	require.Equal(int64(40), cfg.DPOS().FeeFloor(10))

	setConfigRequest.Value = "20"
	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr3)), setConfigRequest)
	require.NoError(err)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	proposal = getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(75), proposal.Percentage)
	require.NotNil(config.Config.Settlement)
	require.Equal(ConfigVoting, config.Config.Status)
	require.Equal("40", config.Config.Settlement.Value)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr4)), setConfigRequest)
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr2)), setConfigRequest)
	require.NoError(err)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: configName,
	})
	proposal = getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(75), proposal.Percentage)
	require.NotNil(config.Config.Settlement)
	require.Equal(ConfigVoting, config.Config.Status)
	require.Equal("40", config.Config.Settlement.Value)

	dposLockTime := "dpos.lockTime"

	err = chainconfigContract.AddConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &AddConfigRequest{
		Names:         []string{dposLockTime},
		VoteThreshold: 30,
		BuildNumber:   0,
	})
	require.NoError(err)

	setConfigRequest = &SetConfigRequest{
		Name:  dposLockTime,
		Value: "1000",
	}

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr4)), setConfigRequest)
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr2)), setConfigRequest)
	require.NoError(err)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: dposLockTime,
	})
	proposal = getMostPopularProposal(config.Config.Proposals)
	require.Equal(uint64(50), proposal.Percentage)
	require.Equal(ConfigVoting, config.Config.Status)
	require.Equal("1000", proposal.Value)

	SetConfigs(contractpb.WrapPluginContext(pctx), 100, 0)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: dposLockTime,
	})
	require.NotNil(config.Config.Settlement)
	require.Equal(ConfigSettled, config.Config.Status)

	configs, err = SetConfigs(contractpb.WrapPluginContext(pctx), 200, 0)
	require.Equal(1, len(configs))
	require.Equal(dposLockTime, configs[0].Name)
	require.Equal("1000", configs[0].Settlement.Value)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), setConfigRequest)
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr3)), setConfigRequest)
	require.NoError(err)

	validators = append(validators[:1], validators[2:]...)
	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: validators,
	})
	require.NoError(err)

	err = chainconfigContract.SetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), setConfigRequest)
	require.NoError(err)

	config, err = chainconfigContract.GetConfig(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetConfigRequest{
		Name: dposLockTime,
	})
	require.Equal(3, len(config.Config.Votes))
}
