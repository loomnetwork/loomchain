package chainconfig

import (
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	dpostypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	plugintypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest          = cctypes.InitRequest
	ListFeaturesRequest  = cctypes.ListFeaturesRequest
	ListFeaturesResponse = cctypes.ListFeaturesResponse

	GetFeatureRequest  = cctypes.GetFeatureRequest
	GetFeatureResponse = cctypes.GetFeatureResponse
	AddFeatureRequest  = cctypes.AddFeatureRequest
	AddFeatureResponse = cctypes.AddFeatureResponse
	SetParamsRequest   = cctypes.SetParamsRequest
	GetParamsRequest   = cctypes.GetParamsRequest
	GetParamsResponse  = cctypes.GetParamsResponse
	Params             = cctypes.Params
	Feature            = cctypes.Feature
	FeatureInfo        = cctypes.FeatureInfo

	UpdateFeatureRequest  = cctypes.UpdateFeatureRequest
	EnableFeatureRequest  = cctypes.EnableFeatureRequest
	EnableFeatureResponse = cctypes.EnableFeatureResponse
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
	ErrFeatureAlreadyExists = errors.New("[ChainConfig] feature already exists")
	// ErrInvalidParams returned if parameters are invalid
	ErrInvalidParams = errors.New("[ChainConfig] invalid params")

	featurePrefix = "feat"
	ownerRole     = "owner"

	addFeatPerm = []byte("addfeat")
	paramsKey   = []byte("params")

	FeaturePending  = cctypes.Feature_PENDING
	FeatureWaiting  = cctypes.Feature_WAITING
	FeatureEnabled  = cctypes.Feature_ENABLED
	FeatureDisabled = cctypes.Feature_DISABLED

	DefaultParam = Params{
		VoteThreshold:         67,
		NumBlockConfirmations: 10,
	}
)

func featureKey(featureName string) []byte {
	return util.PrefixKey([]byte(featurePrefix), []byte(featureName))
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
	ctx.GrantPermissionTo(ownerAddr, addFeatPerm, ownerRole)
	if err := setParams(ctx, req.Params.VoteThreshold, req.Params.NumBlockConfirmations); err != nil {
		return err
	}
	return nil
}

func (c *ChainConfig) SetParams(ctx contract.Context, req *SetParamsRequest) error {
	if ok, _ := ctx.HasPermission(addFeatPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	if req.Params == nil {
		return ErrInvalidRequest
	}

	return setParams(ctx, req.Params.VoteThreshold, req.Params.NumBlockConfirmations)
}

func (c *ChainConfig) GetParams(ctx contract.StaticContext, req *GetParamsRequest) (*GetParamsResponse, error) {
	params, err := getParams(ctx)
	if err != nil {
		return nil, err
	}
	return &GetParamsResponse{
		Params: params,
	}, nil
}

func (c *ChainConfig) FeatureEnabled(ctx contract.StaticContext, req *plugintypes.FeatureEnabledRequest) (*plugintypes.FeatureEnabledResponse, error) {
	val := ctx.FeatureEnabled(req.Name, req.DefaultVal)
	return &plugintypes.FeatureEnabledResponse{
		Value: val,
	}, nil
}

// Enable Feature
func (c *ChainConfig) EnableFeature(ctx contract.Context, req *EnableFeatureRequest) error {
	if req.Name == "" {
		return ErrInvalidRequest
	}
	// check if this is a called from validator
	contractAddr, err := ctx.Resolve("dposV2")
	if err != nil {
		return err
	}
	valsreq := &dpostypes.ListValidatorsRequestV2{}
	var resp dpostypes.ListValidatorsResponseV2
	err = contract.StaticCallMethod(ctx, contractAddr, "ListValidators", valsreq, &resp)
	if err != nil {
		return err
	}

	validators := resp.Statistics
	sender := ctx.Message().Sender

	found := false
	for _, v := range validators {
		if sender.Local.Compare(v.Address.Local) == 0 {
			found = true
		}
	}
	if !found {
		return ErrNotAuthorized
	}

	if err := enableFeature(ctx, req.Name, sender); err != nil {
		return err
	}

	return nil
}

//This method can be called by contract owner only to set known features
func (c *ChainConfig) AddFeature(ctx contract.Context, req *AddFeatureRequest) error {
	if ok, _ := ctx.HasPermission(addFeatPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	if req.Name == "" {
		return ErrInvalidRequest
	}

	if found := ctx.Has(featureKey(req.Name)); found {
		return ErrFeatureAlreadyExists
	}

	feature := Feature{
		Name:   req.Name,
		Status: cctypes.Feature_PENDING,
	}

	if err := ctx.Set(featureKey(req.Name), &feature); err != nil {
		return err
	}

	return nil

}

func (c *ChainConfig) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListFeaturesResponse, error) {
	featureRange := ctx.Range([]byte(featurePrefix))
	listFeaturesResponse := ListFeaturesResponse{
		Features: []*FeatureInfo{},
	}

	for _, m := range featureRange {
		var feature Feature
		if err := proto.Unmarshal(m.Value, &feature); err != nil {
			return nil, errors.Wrap(err, "unmarshal feature")
		}
		featureInfo, err := getFeatureInfo(ctx, feature.Name)
		if err != nil {
			return nil, err
		}
		listFeaturesResponse.Features = append(listFeaturesResponse.Features, featureInfo)
	}

	return &listFeaturesResponse, nil
}

func (c *ChainConfig) GetFeature(ctx contract.StaticContext, req *GetFeatureRequest) (*GetFeatureResponse, error) {
	featureInfo, err := getFeatureInfo(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	getFeatureResponse := GetFeatureResponse{
		FeatureInfo: featureInfo,
	}
	return &getFeatureResponse, nil
}

// EnableFeatures iterates the list of features and change status of feature.
// PENDING feature will be changed to WAITING feature if the percentage of enabled feature validator exceeds the vote threshold.
// WAITING feature will be changed to ENABLED feature after N block confirmation.
// Finally, a list of WAITING features changed to ENABLED features will be returned.
func EnableFeatures(ctx contract.Context, blockHeight uint64) ([]*Feature, error) {
	featureRange := ctx.Range([]byte(featurePrefix))
	features := make([]*Feature, 0)
	params, err := getParams(ctx)
	if err != nil {
		return nil, err
	}

	for _, m := range featureRange {
		var feature Feature
		if err := proto.Unmarshal(m.Value, &feature); err != nil {
			return nil, errors.Wrap(err, "unmarshal feature "+feature.Name)
		}
		featureInfo, err := getFeatureInfo(ctx, feature.Name)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get feature info "+feature.Name)
		}

		switch feature.Status {
		case FeaturePending:
			if featureInfo.Percentage >= params.VoteThreshold {
				feature.Status = FeatureWaiting
				feature.BlockHeight = blockHeight
				if err := ctx.Set(featureKey(feature.Name), &feature); err != nil {
					return nil, err
				}
			}
		case FeatureWaiting:
			if blockHeight > (feature.BlockHeight + params.NumBlockConfirmations) {
				feature.Status = FeatureEnabled
				if err := ctx.Set(featureKey(feature.Name), &feature); err != nil {
					return nil, err
				}
				features = append(features, &feature)
			}
		}
	}
	return features, nil
}

func getFeatureInfo(ctx contract.StaticContext, name string) (*FeatureInfo, error) {
	var feature Feature
	if err := ctx.Get(featureKey(name), &feature); err != nil {
		return nil, err
	}

	// Calculate percentage of validators that enable this feature (only for pending feature)
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
	percentage := uint64((enabledValidatorsCount * 100) / validatorsCount)

	featureInfo := &FeatureInfo{
		Feature:    &feature,
		Percentage: percentage,
	}

	return featureInfo, nil
}

func enableFeature(ctx contract.Context, name string, validator loom.Address) error {
	var feature Feature
	if err := ctx.Get(featureKey(name), &feature); err != nil {
		return err
	}

	found := false
	for _, v := range feature.Validators {
		if validator.Compare(loom.UnmarshalAddressPB(v)) == 0 {
			found = true
			break
		}
	}

	if !found {
		feature.Validators = append(feature.Validators, validator.MarshalPB())
	}

	if err := ctx.Set(featureKey(name), &feature); err != nil {
		return err
	}

	return nil
}

func getParams(ctx contract.StaticContext) (*Params, error) {
	var params Params
	err := ctx.Get(paramsKey, &params)
	if err == contract.ErrNotFound {
		return &DefaultParam, nil
	}
	if err != nil {
		return nil, err
	}
	return &params, nil
}

func setParams(ctx contract.Context, voteThreshold, numBlockConfirmations uint64) error {
	if voteThreshold > 100 {
		return ErrInvalidParams
	}
	params, err := getParams(ctx)
	if err != nil {
		return err
	}
	if voteThreshold != 0 {
		params.VoteThreshold = voteThreshold
	}
	if numBlockConfirmations != 0 {
		params.NumBlockConfirmations = numBlockConfirmations
	}
	return ctx.Set(paramsKey, params)
}

var Contract plugin.Contract = contract.MakePluginContract(&ChainConfig{})
