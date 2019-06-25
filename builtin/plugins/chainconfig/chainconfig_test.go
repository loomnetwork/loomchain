package chainconfig

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain"

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
	buildNumber := uint64(1020)
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

	err = chainconfigContract.SetValidatorInfo(ctx, &SetValidatorInfoRequest{
		BuildNumber: buildNumber,
	})
	require.Error(err, "[ChainConfig] feature not enabled")
	pctx.SetFeature(loomchain.ChainCfgVersion1_2, true)
	err = chainconfigContract.SetValidatorInfo(ctx, &SetValidatorInfoRequest{
		BuildNumber: buildNumber,
	})
	require.NoError(err)
	getValidatorInfo, err := chainconfigContract.GetValidatorInfo(ctx, &GetValidatorInfoRequest{Address: addr1.MarshalPB()})
	require.NoError(err)
	require.Equal(buildNumber, getValidatorInfo.Validator.BuildNumber)
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
	fmt.Println(formatJSON(getFeature))

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
	fmt.Println(formatJSON(getFeature))
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
	fmt.Println(formatJSON(getFeature))

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
	fmt.Println(formatJSON(getFeature))

	featureEnable, err := chainconfigContract.FeatureEnabled(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &plugintypes.FeatureEnabledRequest{
		Name: featureName,
	})

	fmt.Println(formatJSON(featureEnable))
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
