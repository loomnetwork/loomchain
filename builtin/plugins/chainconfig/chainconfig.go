package chainconfig

import (
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	dpostypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	plugintypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"
)

type (
	InitRequest           = cctypes.InitRequest
	ListFeaturesRequest   = cctypes.ListFeaturesRequest
	ListFeaturesResponse  = cctypes.ListFeaturesResponse
	GetFeatureRequest     = cctypes.GetFeatureRequest
	GetFeatureResponse    = cctypes.GetFeatureResponse
	AddFeatureRequest     = cctypes.AddFeatureRequest
	AddFeatureResponse    = cctypes.AddFeatureResponse
	SetParamsRequest      = cctypes.SetParamsRequest
	GetParamsRequest      = cctypes.GetParamsRequest
	GetParamsResponse     = cctypes.GetParamsResponse
	Params                = cctypes.Params
	Feature               = cctypes.Feature
	EnableFeatureRequest  = cctypes.EnableFeatureRequest
	EnableFeatureResponse = cctypes.EnableFeatureResponse
)

const (
	// FeaturePending status indicates a feature hasn't been enabled by majority of validators yet.
	FeaturePending = cctypes.Feature_PENDING
	// FeatureWaiting status indicates a feature has been enabled by majority of validators, but
	// hasn't been activated yet because not enough blocks confirmations have occurred yet.
	FeatureWaiting = cctypes.Feature_WAITING
	// FeatureEnabled status indicates a feature has been enabled by majority of validators, and
	// has been activated on the chain.
	FeatureEnabled = cctypes.Feature_ENABLED
	// FeatureDisabled is not currently used.
	FeatureDisabled = cctypes.Feature_DISABLED
)

var (
	// ErrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[ChainConfig] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[ChainConfig] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[ChainConfig] owner not specified")
	// ErrFeatureAlreadyExists returned if an owner try to set an existing feature
	ErrFeatureAlreadyExists = errors.New("[ChainConfig] feature already exists")
	// ErrInvalidParams returned if parameters are invalid
	ErrInvalidParams = errors.New("[ChainConfig] invalid params")
	// ErrFeatureAlreadyEnabled is returned if a validator tries to enable a feature that's already enabled
	ErrFeatureAlreadyEnabled = errors.New("[ChainConfig] feature already enabled")
	// ErrFeatureNotSupported inidicates that an enabled feature is not supported in the current build
	ErrFeatureNotSupported = errors.New("[Chainconfig] feature is not supported in the current build")
)

const (
	featurePrefix = "ft"
	ownerRole     = "owner"
)

var (
	setParamsPerm  = []byte("setp")
	addFeaturePerm = []byte("addf")

	paramsKey = []byte("params")
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
	ctx.GrantPermissionTo(ownerAddr, setParamsPerm, ownerRole)
	ctx.GrantPermissionTo(ownerAddr, addFeaturePerm, ownerRole)

	for _, feature := range req.Features {
		if feature.Status != FeaturePending && feature.Status != FeatureWaiting {
			return ErrInvalidRequest
		}
		if found := ctx.Has(featureKey(feature.Name)); found {
			return ErrFeatureAlreadyExists
		}
		ctx.Set(featureKey(feature.Name), feature)
	}

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

	if ok, _ := ctx.HasPermission(setParamsPerm, []string{ownerRole}); !ok {
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
// The feature won't actually become active until the majority of the validators have indicated
// they're ready.
func (c *ChainConfig) EnableFeature(ctx contract.Context, req *EnableFeatureRequest) error {
	if len(req.Names) == 0 {
		return ErrInvalidRequest
	}
	for _, name := range req.Names {
		if err := enableFeature(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// AddFeature should be called by the contract owner to add a new feature the validators can enable.
func (c *ChainConfig) AddFeature(ctx contract.Context, req *AddFeatureRequest) error {
	if len(req.Name) == 0 {
		return ErrInvalidRequest
	}
	if err := addFeature(ctx, req.Name, req.BuildNumber); err != nil {
		return err
	}
	return nil
}

// ListFeatures returns info about all the currently known features.
func (c *ChainConfig) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListFeaturesResponse, error) {
	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return nil, err
	}

	featureRange := ctx.Range([]byte(featurePrefix))
	features := []*Feature{}
	for _, m := range featureRange {
		var f Feature
		if err := proto.Unmarshal(m.Value, &f); err != nil {
			return nil, errors.Wrapf(err, "unmarshal feature %s", string(m.Key))
		}
		feature, err := getFeature(ctx, f.Name, curValidators)
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

	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return nil, err
	}

	feature, err := getFeature(ctx, req.Name, curValidators)
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
// Returns a list of features whose status has changed from WAITING to ENABLED at the given height.
func EnableFeatures(ctx contract.Context, blockHeight uint64, buildNumber uint64) ([]*Feature, error) {
	params, err := getParams(ctx)
	if err != nil {
		return nil, err
	}

	curValidators, err := getCurrentValidators(ctx)
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
		// this one will calculate the percentage for pending feature
		feature, err := getFeature(ctx, f.Name, curValidators)
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
				ctx.Logger().Info(
					"[Feature status changed]",
					"name", feature.Name,
					"from", FeaturePending,
					"to", FeatureWaiting,
					"block_height", blockHeight,
					"percentage", feature.Percentage,
				)
			}
		case FeatureWaiting:
			if buildNumber < feature.BuildNumber {
				ctx.Logger().Warn(
					"[Unsupported feature is going to be enabled]",
					"name", feature.Name,
					"minimum-build", feature.BuildNumber,
					"current-build", buildNumber,
					"block_height", blockHeight,
					"percentage", feature.Percentage,
				)
			}
			if blockHeight > (feature.BlockHeight + params.NumBlockConfirmations) {
				if buildNumber < feature.BuildNumber {
					return nil, ErrFeatureNotSupported
				}
				feature.Status = FeatureEnabled
				if err := ctx.Set(featureKey(feature.Name), feature); err != nil {
					return nil, err
				}
				enabledFeatures = append(enabledFeatures, feature)
				ctx.Logger().Info(
					"[Feature status changed]",
					"name", feature.Name,
					"from", FeatureWaiting,
					"to", FeatureEnabled,
					"block_height", blockHeight,
					"percentage", feature.Percentage,
				)
			}
		}

	}
	return enabledFeatures, nil
}

func getCurrentValidators(ctx contract.StaticContext) ([]loom.Address, error) {
	// TODO: Replace all this with ctx.Validators() when it's hooked up to DPOSv3 (and ideally DPOSv2)
	if ctx.FeatureEnabled(loomchain.DPOSVersion3Feature, false) {
		contractAddr, err := ctx.Resolve("dposV3")
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve address of DPOSv3 contract")
		}

		req := &dposv3.ListValidatorsRequest{}
		var resp dposv3.ListValidatorsResponse
		if err := contract.StaticCallMethod(ctx, contractAddr, "ListValidators", req, &resp); err != nil {
			return nil, errors.Wrap(err, "failed to call ListValidators")
		}

		validators := make([]loom.Address, 0, len(resp.Statistics))
		for _, v := range resp.Statistics {
			if v != nil {
				addr := loom.UnmarshalAddressPB(v.Address)
				validators = append(validators, addr)
			}
		}
		return validators, nil
	}

	// Fallback to DPOSv2 if DPOSv3 isn't enabled
	contractAddr, err := ctx.Resolve("dposV2")
	if err != nil {
		// No DPOSv2 either? Fine, then features can only be enabled via the contract genesis!
		if err == registry.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to resolve address of DPOS contract")
	}

	req := &dpostypes.ListValidatorsRequestV2{}
	var resp dpostypes.ListValidatorsResponseV2
	if err := contract.StaticCallMethod(ctx, contractAddr, "ListValidatorsSimple", req, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to call ListValidators")
	}

	validators := make([]loom.Address, 0, len(resp.Statistics))
	for _, v := range resp.Statistics {
		if v != nil {
			addr := loom.UnmarshalAddressPB(v.Address)
			validators = append(validators, addr)
		}
	}
	return validators, nil
}

func getFeature(ctx contract.StaticContext, name string, curValidators []loom.Address) (*Feature, error) {
	var feature Feature
	if err := ctx.Get(featureKey(name), &feature); err != nil {
		return nil, err
	}

	if feature.Status != FeaturePending {
		return &feature, nil
	}

	// Calculate percentage of validators that enabled this pending feature so far
	enabledValidatorsCount := 0
	validatorsHashMap := map[string]bool{}

	for _, v := range curValidators {
		validatorsHashMap[v.Local.String()] = false
	}
	for _, v := range feature.Validators {
		validatorsHashMap[v.Local.String()] = true
	}
	for _, v := range validatorsHashMap {
		if v {
			enabledValidatorsCount++
		}
	}
	if len(curValidators) > 0 {
		feature.Percentage = uint64((enabledValidatorsCount * 100) / len(curValidators))
	}
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

func enableFeature(ctx contract.Context, name string) error {
	if name == "" {
		return ErrInvalidRequest
	}

	// check if this is a called from validator
	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return err
	}
	sender := ctx.Message().Sender

	found := false
	for _, v := range curValidators {
		if sender.Compare(v) == 0 {
			found = true
			break
		}
	}
	if !found {
		return ErrNotAuthorized
	}

	// record the fact that the validator is ready to enable the feature
	var feature Feature
	if err := ctx.Get(featureKey(name), &feature); err != nil {
		return errors.Wrapf(err, "feature '%s' not found", name)
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

	return ctx.Set(featureKey(name), &feature)
}

func addFeature(ctx contract.Context, name string, buildNumber uint64) error {
	if name == "" {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(addFeaturePerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	if found := ctx.Has(featureKey(name)); found {
		return ErrFeatureAlreadyExists
	}

	feature := Feature{
		Name:        name,
		BuildNumber: buildNumber,
		Status:      FeaturePending,
	}

	if err := ctx.Set(featureKey(name), &feature); err != nil {
		return err
	}

	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&ChainConfig{})
