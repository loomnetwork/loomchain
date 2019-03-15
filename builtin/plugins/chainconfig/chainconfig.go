package chainconfig

import (
	"fmt"
	"math"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	chainconfigtypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	dpostypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest          = chainconfig.ChainConfigInitRequest
	ListFeaturesRequest  = chainconfig.ListFeaturesRequest
	ListFeaturesResponse = chainconfig.ListFeaturesResponse

	GetFeatureRequest  = chainconfig.GetFeatureRequest
	GetFeatureResponse = chainconfig.GetFeatureResponse
	SetFeatureRequest  = chainconfig.SetFeatureRequest
	SetFeatureResponse = chainconfig.SetFeatureResponse
	Feature            = chainconfig.Feature
	Config             = chainconfig.Config

	UpdateFeatureRequest  = chainconfig.UpdateFeatureRequest
	EnableFeatureRequest  = chainconfig.EnableFeatureRequest
	EnableFeatureResponse = chainconfig.EnableFeatureResponse
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[ChainConfig] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[ChainConfig] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[ChainConfig] owner not specified")
	// ErrFeatureFound returned if an owner try to set an existing feature
	ErrFeatureFound = errors.New("[ChainConfig] feature found")

	configPrefix  = "config-"
	featurePrefix = "feature-"
	ownerRole     = "owner"

	submitKnownFeaturePerm = []byte("submit-known-feature")

	//feature status
	pending  = "pending"
	disabled = "disabled"
	enabled  = "enabled"
)

func configKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(configPrefix), addr.Bytes())
}

type ChainConfig struct {
}

func (c *ChainConfig) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "chainconfig",
		Version: "1.0.0",
	}, nil
}

func (c *ChainConfig) Init(ctx contract.Context, req *InitRequest) error {
	if req.Owner == nil {
		return ErrOwnerNotSpecified
	}
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	ctx.GrantPermissionTo(ownerAddr, submitKnownFeaturePerm, ownerRole)
	return nil
}

//TODO: require manual sign by all the validators, its fairly safe, for now we allow one way movement
//worst thing would be a consensus issue

//TODO: first pass only has features, which are a subset of configs
//that are only boolean

// Enable Feature
func (c *ChainConfig) EnableFeature(ctx contract.Context, req *EnableFeatureRequest) (*EnableFeatureResponse, error) {
	// check if this is a called from validator
	contractAddr, err := ctx.Resolve("dposV2")
	if err != nil {
		return nil, err
	}
	valsreq := &dpostypes.ListValidatorsRequestV2{}
	var resp dpostypes.ListValidatorsResponseV2
	err = contract.StaticCallMethod(ctx, contractAddr, "ListValidators", valsreq, &resp)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)

	validators := resp.Statistics
	sender := ctx.Message().Sender

	found := false
	for _, v := range validators {
		if sender.Local.Compare(v.Address.Local) == 0 {
			found = true
		}
	}
	if !found {
		return nil, ErrNotAuthorized
	}

	if err := enableFeature(ctx, req.Name, &sender); err != nil {
		return nil, err
	}

	featureResponse, err := getFeatureResponse(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	enableFeatureResponse := EnableFeatureResponse{
		Feature: featureResponse,
	}

	return &enableFeatureResponse, nil
}

//This method can be called by contract owner only to set known features
func (c *ChainConfig) SetFeature(ctx contract.Context, req *SetFeatureRequest) (*SetFeatureResponse, error) {
	if ok, _ := ctx.HasPermission(submitKnownFeaturePerm, []string{ownerRole}); !ok {
		return nil, ErrNotAuthorized
	}

	if found := ctx.Has([]byte(featurePrefix + req.Name)); found {
		return nil, ErrFeatureFound
	}

	var validators []*types.Address

	feature := Feature{
		Name:       req.Name,
		Enabled:    "false",
		Status:     pending,
		Validators: validators,
	}

	if err := ctx.Set([]byte(featurePrefix+req.Name), &feature); err != nil {
		return nil, err
	}

	featureResponse, err := getFeatureResponse(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	resp := SetFeatureResponse{
		Feature: featureResponse,
	}

	return &resp, nil

}

func (c *ChainConfig) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListFeaturesResponse, error) {
	featureRange := ctx.Range([]byte(featurePrefix))
	listFeaturesResponse := ListFeaturesResponse{
		Features: []*GetFeatureResponse{},
	}

	for _, m := range featureRange {
		var feature Feature
		if err := proto.Unmarshal(m.Value, &feature); err != nil {
			return &ListFeaturesResponse{}, errors.Wrap(err, "unmarshal feature")
		}
		featureResponse, err := getFeatureResponse(ctx, feature.Name)
		if err != nil {
			return nil, err
		}
		listFeaturesResponse.Features = append(listFeaturesResponse.Features, featureResponse)
	}

	return &listFeaturesResponse, nil
}

func (c *ChainConfig) GetFeature(ctx contract.StaticContext, req *GetFeatureRequest) (*GetFeatureResponse, error) {
	featureResponse, err := getFeatureResponse(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return featureResponse, nil
}

func getFeatureResponse(ctx contract.StaticContext, name string) (*GetFeatureResponse, error) {
	var feature chainconfigtypes.Feature
	if err := ctx.Get([]byte(featurePrefix+name), &feature); err != nil {
		return nil, err
	}

	// Calculate percentage of validators that enable this feature
	contractAddr, err := ctx.Resolve("dposV2")
	if err != nil {
		return nil, err
	}
	valsreq := &dpostypes.ListValidatorsRequestV2{}
	var resp dpostypes.ListValidatorsResponseV2
	if err = contract.StaticCallMethod(ctx, contractAddr, "ListValidators", valsreq, &resp); err != nil {
		return nil, err
	}

	validators := resp.Statistics
	validatorsCount := len(validators)
	enabledValidatorsCount := 0
	validatorsHashMap := map[string]bool{}

	for _, v := range validators {
		validatorsHashMap[v.Address.Local.String()] = false
	}
	for _, v := range feature.Validators {
		validatorsHashMap[v.Local.String()] = true
	}
	for _, v := range validatorsHashMap {
		if v {
			enabledValidatorsCount++
		}
	}
	preConvert := math.Round((float64(enabledValidatorsCount) / float64(validatorsCount)) * 100)
	percentage := uint64(preConvert)

	featureResponse := &GetFeatureResponse{
		Feature:    &feature,
		Percentage: percentage,
	}

	return featureResponse, nil
}

func enableFeature(ctx contract.Context, name string, validator *loom.Address) error {

	var feature chainconfigtypes.Feature
	if err := ctx.Get([]byte(featurePrefix+name), &feature); err != nil {
		return err
	}

	found := false
	for _, v := range feature.Validators {
		if validator.Compare(loom.UnmarshalAddressPB(v)) == 0 {
			found = true
		}
	}

	if !found {
		feature.Validators = append(feature.Validators, validator.MarshalPB())
	}

	if err := ctx.Set([]byte(featurePrefix+name), &feature); err != nil {
		return err
	}

	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&ChainConfig{})
