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
	// ErrFeatureAlreadyEnabled is returned if a validator tries to enable a feature that's already enabled
	ErrFeatureAlreadyEnabled = errors.New("[ChainConfig] feature already enabled")

	featurePrefix = "feat"
	ownerRole     = "owner"

	addFeatPerm = []byte("addfeat")
	paramsKey   = []byte("params")

	FeaturePending  = cctypes.Feature_PENDING
	FeatureWaiting  = cctypes.Feature_WAITING
	FeatureEnabled  = cctypes.Feature_ENABLED
	FeatureDisabled = cctypes.Feature_DISABLED
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

	if req.Params != nil {
		if err := setParams(ctx, req.Params.VoteThreshold, req.Params.NumBlockConfirmations); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainConfig) SetParams(ctx contract.Context, req *SetParamsRequest) error {
	if req.Params == nil {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(addFeatPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
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

// FeatureEnabled checks if a specific feature is currently enabled on the chain, which means that
// it has been enabled by a sufficient number of validators, and has been activated.
func (c *ChainConfig) FeatureEnabled(
	ctx contract.StaticContext, req *plugintypes.FeatureEnabledRequest,
) (*plugintypes.FeatureEnabledResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidRequest
	}

	val := ctx.FeatureEnabled(req.Name, req.DefaultVal)
	return &plugintypes.FeatureEnabledResponse{
		Value: val,
	}, nil
}

// EnableFeature should be called by a validator to indicate they're ready to activate a feature.
// The feature won't actually become active until a sufficient number of validators have indicated
// they're ready.
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
	if err := contract.StaticCallMethod(ctx, contractAddr, "ListValidators", valsreq, &resp); err != nil {
		return errors.Wrap(err, "failed to call ListValidators")
	}

	validators := resp.Statistics
	sender := ctx.Message().Sender

	found := false
	for _, v := range validators {
		if sender.Local.Compare(v.Address.Local) == 0 {
			found = true
			break
		}
	}
	if !found {
		return ErrNotAuthorized
	}

	// record the fact that the validator is ready to enable the feature
	var feature Feature
	if err := ctx.Get(featureKey(req.Name), &feature); err != nil {
		return errors.Wrapf(err, "feature '%s' not found", req.Name)
	}

	// if the feature has already been activated there's no point in recording additional votes
	if feature.Status == FeatureEnabled {
		return ErrFeatureAlreadyEnabled
	}

	for _, v := range feature.Validators {
		if sender.Compare(loom.UnmarshalAddressPB(v)) == 0 {
			return ErrFeatureAlreadyEnabled
		}
	}

	feature.Validators = append(feature.Validators, sender.MarshalPB())

	return ctx.Set(featureKey(req.Name), &feature)
}

// AddFeature should be called by the contract owner to add a new feature the validators can enable.
func (c *ChainConfig) AddFeature(ctx contract.Context, req *AddFeatureRequest) error {
	if req.Name == "" {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(addFeatPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	if found := ctx.Has(featureKey(req.Name)); found {
		return ErrFeatureAlreadyExists
	}

	feature := Feature{
		Name:   req.Name,
		Status: FeaturePending,
	}

	if err := ctx.Set(featureKey(req.Name), &feature); err != nil {
		return err
	}

	return nil
}

// ListFeatures returns info about all the currently known features.
func (c *ChainConfig) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListFeaturesResponse, error) {
	featureRange := ctx.Range([]byte(featurePrefix))
	features := []*Feature{}
	for _, m := range featureRange {
		var f Feature
		if err := proto.Unmarshal(m.Value, &f); err != nil {
			return nil, errors.Wrapf(err, "unmarshal feature %s", string(m.Key))
		}
		feature, err := getFeature(ctx, f.Name)
		if err != nil {
			return nil, err
		}
		features = append(features, feature)
	}

	return &ListFeaturesResponse{
		Features: features,
	}, nil
}

// GetFeature returns info about a specific feature.
func (c *ChainConfig) GetFeature(ctx contract.StaticContext, req *GetFeatureRequest) (*GetFeatureResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidRequest
	}

	feature, err := getFeature(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &GetFeatureResponse{
		Feature: feature,
	}, nil
}

// EnableFeatures updates the status of features that haven't been activated yet:
// - A PENDING feature will become WAITING once the percentage of validators that have enabled the
//   feature reaches a certain threshold.
// - A WAITING feature will become ENABLED after a sufficient number of block confirmations.
//
// Returns a list of features whose status has changed from WAITING to ENABLED at the given height.
func EnableFeatures(ctx contract.Context, blockHeight uint64) ([]*Feature, error) {
	params, err := getParams(ctx)
	if err != nil {
		return nil, err
	}

	featureRange := ctx.Range([]byte(featurePrefix))
	enabledFeatures := make([]*Feature, 0)
	for _, m := range featureRange {
		var f Feature
		if err := proto.Unmarshal(m.Value, &f); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal feature %s", string(m.Key))
		}
		//this one will calculate the percentage for pending feature
		feature, err := getFeature(ctx, f.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get feature info %s", f.Name)
		}

		switch feature.Status {
		case FeaturePending:
			if feature.Percentage >= params.VoteThreshold {
				feature.Status = FeatureWaiting
				feature.BlockHeight = blockHeight
				if err := ctx.Set(featureKey(feature.Name), feature); err != nil {
					return nil, err
				}
			}
		case FeatureWaiting:
			if blockHeight > (feature.BlockHeight + params.NumBlockConfirmations) {
				feature.Status = FeatureEnabled
				if err := ctx.Set(featureKey(feature.Name), feature); err != nil {
					return nil, err
				}
				enabledFeatures = append(enabledFeatures, feature)
			}
		}
	}
	return enabledFeatures, nil
}

func getFeature(ctx contract.StaticContext, name string) (*Feature, error) {
	var feature Feature
	if err := ctx.Get(featureKey(name), &feature); err != nil {
		return nil, err
	}

	if feature.Status != FeaturePending {
		return &feature, nil
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
	feature.Percentage = uint64((enabledValidatorsCount * 100) / validatorsCount)
	return &feature, nil
}

func getParams(ctx contract.StaticContext) (*Params, error) {
	var params Params
	err := ctx.Get(paramsKey, &params)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrap(err, "failed to load chainconfig params")
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
