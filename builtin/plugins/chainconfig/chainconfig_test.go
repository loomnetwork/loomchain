package chainconfig

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
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
	encoder := base64.StdEncoding
	pubKeyB64_1, _ := encoder.DecodeString(pubKey1)
	addr1 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	//setup dposv2 fake contract
	pctx := plugin.CreateFakeContext(addr1, addr1)

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
	varlidators := []*dposv2.Validator{
		&dposv2.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
	}
	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: varlidators,
	})
	require.NoError(err)

	//setup chainconfig contract
	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(ctx, &AddFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	getFeature, err := chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.FeatureInfo.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.FeatureInfo.Feature.Status)
	require.Equal(uint64(0), getFeature.FeatureInfo.Percentage)

	listFeatures, err := chainconfigContract.ListFeatures(ctx, &ListFeaturesRequest{})
	require.NoError(err)
	require.Equal(1, len(listFeatures.FeatureInfos))

	err = chainconfigContract.EnableFeature(ctx, &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	getFeature, err = chainconfigContract.GetFeature(ctx, &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.FeatureInfo.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.FeatureInfo.Feature.Status)
	require.Equal(uint64(100), getFeature.FeatureInfo.Percentage)
}

func (c *ChainConfigTestSuite) TestPermission() {
	require := c.Require()
	featureName := "hardfork"
	encoder := base64.StdEncoding
	pubKeyB64_1, _ := encoder.DecodeString(pubKey1)
	pubKeyB64_2, _ := encoder.DecodeString(pubKey2)
	addr1 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_1)}
	addr2 := loom.Address{ChainID: "", Local: loom.LocalAddressFromPublicKey(pubKeyB64_2)}
	//setup dposv2 fake contract
	pctx := plugin.CreateFakeContext(addr1, addr1)

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
	varlidators := []*dposv2.Validator{
		&dposv2.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
	}
	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: varlidators,
	})
	require.NoError(err)

	//setup chainconfig contract
	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(ctx, &AddFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &AddFeatureRequest{
		Name: "newFeature",
	})
	require.Equal(ErrNotAuthorized, err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.Equal(ErrNotAuthorized, err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
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
	varlidators := []*dposv2.Validator{
		&dposv2.Validator{
			PubKey: pubKeyB64_1,
			Power:  10,
		},
		&dposv2.Validator{
			PubKey: pubKeyB64_2,
			Power:  10,
		},
		&dposv2.Validator{
			PubKey: pubKeyB64_3,
			Power:  10,
		},
		&dposv2.Validator{
			PubKey: pubKeyB64_4,
			Power:  10,
		},
	}
	err = dposv2Contract.Init(ctx, &dposv2.InitRequest{
		Params: &dposv2.Params{
			ValidatorCount: 21,
		},
		Validators: varlidators,
	})
	require.NoError(err)

	chainconfigContract := &ChainConfig{}
	err = chainconfigContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
	})
	require.NoError(err)

	err = chainconfigContract.AddFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &AddFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr4)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	getFeature, err := chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr1)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.FeatureInfo.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.FeatureInfo.Feature.Status)
	require.Equal(uint64(50), getFeature.FeatureInfo.Percentage)
	fmt.Println(formatJSON(getFeature))

	err = chainconfigContract.EnableFeature(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &EnableFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)

	getFeature, err = chainconfigContract.GetFeature(contractpb.WrapPluginContext(pctx.WithSender(addr2)), &GetFeatureRequest{
		Name: featureName,
	})
	require.NoError(err)
	require.Equal(featureName, getFeature.FeatureInfo.Feature.Name)
	require.Equal(cctypes.Feature_PENDING, getFeature.FeatureInfo.Feature.Status)
	require.Equal(uint64(75), getFeature.FeatureInfo.Percentage)
	fmt.Println(formatJSON(getFeature))
}
