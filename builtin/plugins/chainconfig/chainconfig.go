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
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
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
	RemoveFeatureRequest  = cctypes.RemoveFeatureRequest
	SetParamsRequest      = cctypes.SetParamsRequest
	GetParamsRequest      = cctypes.GetParamsRequest
	GetParamsResponse     = cctypes.GetParamsResponse
	Params                = cctypes.Params
	Feature               = cctypes.Feature
	EnableFeatureRequest  = cctypes.EnableFeatureRequest
	EnableFeatureResponse = cctypes.EnableFeatureResponse

	Config              = cctypes.Config
	Vote                = cctypes.Vote
	Proposal            = cctypes.Proposal
	AddConfigRequest    = cctypes.AddConfigRequest
	GetConfigRequest    = cctypes.GetConfigRequest
	GetConfigResponse   = cctypes.GetConfigResponse
	ListConfigsRequest  = cctypes.ListConfigsRequest
	ListConfigsResponse = cctypes.ListConfigsResponse
	SetConfigRequest    = cctypes.SetConfigRequest
	ConfigValueRequest  = cctypes.ConfigValueRequest
	ConfigValueResponse = cctypes.ConfigValueResponse
	RemoveConfigRequest = cctypes.RemoveConfigRequest

	ValidatorInfo              = cctypes.ValidatorInfo
	GetValidatorInfoRequest    = cctypes.GetValidatorInfoRequest
	GetValidatorInfoResponse   = cctypes.GetValidatorInfoResponse
	SetValidatorInfoRequest    = cctypes.SetValidatorInfoRequest
	ListValidatorsInfoRequest  = cctypes.ListValidatorsInfoRequest
	ListValidatorsInfoResponse = cctypes.ListValidatorsInfoResponse
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

	// ConfigVoting status indicates new config settings are being voted.
	ConfigVoting = cctypes.Config_VOTING
	// ConfigSettled status indicates a new config setting has been settled by majority of validators, but
	// hasn't been activated yet because not enough blocks confirmations have occurred yet.
	ConfigSettled = cctypes.Config_SETTLED
	// ConfigActivated status indicates a new config has been voted by majority of validators, and
	// has been activated on the chain.
	ConfigActivated = cctypes.Config_ACTIVATED
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
	// ErrEmptyValidatorsList is returned if ctx.Validators() return empty validators list.
	ErrEmptyValidatorsList = errors.New("[ChainConfig] empty validators list")
	// ErrFeatureNotSupported inidicates that an enabled feature is not supported in the current build
	ErrFeatureNotSupported = errors.New("[ChainConfig] feature is not supported in the current build")
	// ErrFeatureNotFound indicates that a feature does not exist
	ErrFeatureNotFound = errors.New("[ChainConfig] feature not found")
	// ErrFeatureNotEnabled indacates that a feature has not been enabled
	// by majority of validators, and has not been activated on the chain.
	ErrFeatureNotEnabled = errors.New("[ChainConfig] feature not enabled")

	// ErrConfigNotFound indicates that a config does not exist
	ErrConfigNotFound = errors.New("[ChainConfig] config not found")
	// ErrConfigNotSupported inidicates that an enabled config is not supported in the current build
	ErrConfigNotSupported = errors.New("[ChainConfig] config is not supported in the current build")
	// ErrConfigAlreadyExists returned if an owner try to set an existing config
	ErrConfigAlreadyExists = errors.New("[ChainConfig] config already exists")
	// ErrConfigAlreadySettled is returned if a validator tries to vote a confg that's already settled
	ErrConfigAlreadySettled = errors.New("[ChainConfig] config already settled")
	// ErrConfigNonVotable is returned if a validator tries to vote a non-votable config
	ErrConfigNonVotable = errors.New("[ChainConfig] config is not votable")
)

const (
	featurePrefix       = "ft"
	configPrefix        = "cfg"
	ownerRole           = "owner"
	validatorInfoPrefix = "vi"
)

var (
	setParamsPerm  = []byte("setp")
	addFeaturePerm = []byte("addf")

	paramsKey = []byte("params")
)

func featureKey(featureName string) []byte {
	return util.PrefixKey([]byte(featurePrefix), []byte(featureName))
}

func configKey(configName string) []byte {
	return util.PrefixKey([]byte(configPrefix), []byte(configName))
}

func validatorInfoKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(validatorInfoPrefix), addr.Bytes())
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

// AddFeature should be called by the contract owner to add new features the validators can enable.
func (c *ChainConfig) AddFeature(ctx contract.Context, req *AddFeatureRequest) error {
	if len(req.Names) == 0 {
		return ErrInvalidRequest
	}
	for _, name := range req.Names {
		if err := addFeature(ctx, name, req.BuildNumber, req.AutoEnable); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFeature should be called by the contract owner to remove features.
// NOTE: Features can only be removed before they're activated by the chain.
func (c *ChainConfig) RemoveFeature(ctx contract.Context, req *RemoveFeatureRequest) error {
	if len(req.Names) == 0 {
		return ErrInvalidRequest
	}
	for _, name := range req.Names {
		if err := removeFeature(ctx, name); err != nil {
			return err
		}
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
func EnableFeatures(ctx contract.Context, blockHeight, buildNumber uint64) ([]*Feature, error) {
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

// AddConfig should be called by the contract owner to add new config the validators can vote.
func (c *ChainConfig) AddConfig(ctx contract.Context, req *AddConfigRequest) error {
	if !ctx.FeatureEnabled(loomchain.ChainCfgVersion1_3, false) {
		return ErrFeatureNotEnabled
	}

	for _, name := range req.Names {
		if name == "" {
			return ErrInvalidRequest
		}
	}

	// TODO: config should have its own permission
	if ok, _ := ctx.HasPermission(addFeaturePerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	for _, name := range req.Names {
		if found := ctx.Has(configKey(name)); found {
			return ErrConfigAlreadyExists
		}

		config := Config{
			Name:          name,
			BuildNumber:   req.BuildNumber,
			Status:        ConfigVoting,
			VoteThreshold: req.VoteThreshold,
		}

		if config.VoteThreshold == 0 {
			sender := ctx.Message().Sender
			vote := &Vote{
				Validator: sender.MarshalPB(),
				Value:     req.Value,
			}
			config.Votes = []*Vote{vote}
		}

		if err := ctx.Set(configKey(name), &config); err != nil {
			return err
		}
	}

	return nil
}

// GetConfig returns info about a specific config.
func (c *ChainConfig) GetConfig(ctx contract.StaticContext, req *GetConfigRequest) (*GetConfigResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidRequest
	}

	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return nil, err
	}

	config, err := getConfig(ctx, req.Name, curValidators)
	if err != nil {
		return nil, err
	}

	return &GetConfigResponse{
		Config: config,
	}, nil
}

// SetConfig should be called by a validator to indicate they want to propose a new config value.
func (c *ChainConfig) SetConfig(ctx contract.Context, req *SetConfigRequest) error {
	if !ctx.FeatureEnabled(loomchain.ChainCfgVersion1_3, false) {
		return ErrFeatureNotEnabled
	}

	if req.Name == "" || req.Value == "" {
		return ErrInvalidRequest
	}

	// check if this is a called from a validator
	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return err
	}

	// create validators hash map for checking valid validator
	curValidatorsHashMap := make(map[string]bool, 0)
	for _, validator := range curValidators {
		curValidatorsHashMap[validator.String()] = true
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

	// record the fact that the validator is ready to set the config
	var config Config
	if err := ctx.Get(configKey(req.Name), &config); err != nil {
		return errors.Wrapf(err, "config '%s' not found", req.Name)
	}

	// only the contract owner can set value of non-votable config
	contractOwner, _ := ctx.HasPermission(addFeaturePerm, []string{ownerRole})
	if config.VoteThreshold == 0 && !contractOwner {
		return ErrConfigNonVotable
	}

	// if the config has already been settled there's no point in recording additional votes
	if config.Status == ConfigSettled {
		return ErrConfigAlreadySettled
		// if a validator sets a config that has been activated, change config status to voting,
	} else if config.Status == ConfigActivated {
		config.Status = ConfigVoting
	}

	votes := make([]*Vote, 0)
	var vote *Vote
	for _, v := range config.Votes {
		// only add valid votes to vote list
		voterAddr := loom.UnmarshalAddressPB(v.Validator).String()
		if curValidatorsHashMap[voterAddr] {
			votes = append(votes, v)
		}

		// find the vote of this validator
		if sender.Compare(loom.UnmarshalAddressPB(v.Validator)) == 0 {
			vote = v
		}
	}

	if vote == nil {
		vote := &Vote{
			Validator: sender.MarshalPB(),
			Value:     req.Value,
		}
		votes = append(votes, vote)
	} else {
		vote.Value = req.Value
	}

	config.Votes = votes

	return ctx.Set(configKey(req.Name), &config)
}

// RemoveConfig should be called by the contract owner to remove configs.
func (c *ChainConfig) RemoveConfig(ctx contract.Context, req *RemoveConfigRequest) error {
	if !ctx.FeatureEnabled(loomchain.ChainCfgVersion1_3, false) {
		return ErrFeatureNotEnabled
	}

	if len(req.Names) == 0 {
		return ErrInvalidRequest
	}
	for _, name := range req.Names {
		if name == "" {
			return ErrInvalidRequest
		}
		// TODO: config should have its own permission
		if ok, _ := ctx.HasPermission(addFeaturePerm, []string{ownerRole}); !ok {
			return ErrNotAuthorized
		}
		if found := ctx.Has(configKey(name)); !found {
			return ErrConfigNotFound
		}
		ctx.Delete(configKey(name))
	}
	return nil
}

// ConfigValue checks value of a specific config that is currently set on the chain, which means that
// it has been voted by a sufficient number of validators, and has been set.
func (c *ChainConfig) ConfigValue(ctx contract.StaticContext, req *ConfigValueRequest) (*ConfigValueResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidRequest
	}
	cfg := ctx.ChainConfig()
	val := cfg.GetConfig(req.Name)
	return &ConfigValueResponse{
		Name:  req.Name,
		Value: val,
	}, nil
}

// ListConfigs returns info about all the currently known configs.
func (c *ChainConfig) ListConfigs(ctx contract.StaticContext, req *ListConfigsRequest) (*ListConfigsResponse, error) {
	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return nil, err
	}

	configRange := ctx.Range([]byte(configPrefix))
	configs := []*Config{}
	for _, m := range configRange {
		var cfg Config
		if err := proto.Unmarshal(m.Value, &cfg); err != nil {
			return nil, errors.Wrapf(err, "unmarshal config %s", string(m.Key))
		}
		config, err := getConfig(ctx, cfg.Name, curValidators)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return &ListConfigsResponse{
		Configs: configs,
	}, nil
}

// SetConfigs updates the status of configs that haven't been activated yet:
// - A VOTING config will become SETTLED once the percentage of validators that have voted to set
//   a particular value reaches a certain threshold.
// - A SETTLED config will become ACTIVATED after a sufficient number of block confirmations.
// Returns a list of configs whose status has changed from SETTLED to ACTIVTED at the given height.
func SetConfigs(ctx contract.Context, blockHeight, buildNumber uint64) ([]*Config, error) {
	params, err := getParams(ctx)
	if err != nil {
		return nil, err
	}

	curValidators, err := getCurrentValidators(ctx)
	if err != nil {
		return nil, err
	}

	configRange := ctx.Range([]byte(configPrefix))
	activatedConfigs := make([]*Config, 0)
	for _, data := range configRange {
		var cfg Config
		if err := proto.Unmarshal(data.Value, &cfg); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal config %s", string(data.Key))
		}
		// this one will calculate the percentage for voting config
		config, err := getConfig(ctx, cfg.Name, curValidators)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get config info %s", cfg.Name)
		}

		switch config.Status {
		case ConfigVoting:
			proposal := getMostPopularProposal(config.Proposals)
			if proposal == nil {
				continue
			}
			if proposal.Percentage >= config.VoteThreshold {
				config.Status = ConfigSettled
				config.BlockHeight = blockHeight
				config.Settlement = proposal
				if err := ctx.Set(configKey(config.Name), config); err != nil {
					return nil, err
				}
				ctx.Logger().Info(
					"[Config status changed]",
					"name", config.Name,
					"from", ConfigVoting,
					"to", ConfigSettled,
					"block_height", blockHeight,
					"percentage", proposal.Percentage,
					"value", proposal.Value,
				)
			}
		case ConfigSettled:
			if blockHeight > (config.BlockHeight + params.NumBlockConfirmations) {
				if buildNumber < config.BuildNumber {
					return nil, ErrConfigNotSupported
				}
				config.Status = ConfigActivated
				if err := ctx.Set(configKey(config.Name), config); err != nil {
					return nil, err
				}
				activatedConfigs = append(activatedConfigs, config)
				ctx.Logger().Info(
					"[Config status changed]",
					"name", config.Name,
					"from", ConfigSettled,
					"to", ConfigActivated,
					"block_height", blockHeight,
					"percentage", config.Settlement.Percentage,
					"value", config.Settlement.Value,
				)
			}
		}

	}
	return activatedConfigs, nil
}

// ConfigList returns the list of configs on the chainconfig contract
func ConfigList(ctx contract.Context) ([]*Config, error) {
	configRange := ctx.Range([]byte(configPrefix))
	configList := make([]*Config, 0)
	for _, data := range configRange {
		var config Config
		if err := proto.Unmarshal(data.Value, &config); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal config %s", string(data.Key))
		}
		configList = append(configList, &config)
	}
	return configList, nil
}

func getCurrentValidatorsFromDPOS(ctx contract.StaticContext) ([]loom.Address, error) {
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

	if len(validators) == 0 {
		return nil, ErrEmptyValidatorsList
	}

	return validators, nil
}

func getCurrentValidators(ctx contract.StaticContext) ([]loom.Address, error) {
	if !ctx.FeatureEnabled(loomchain.ChainCfgVersion1_1, false) {
		return getCurrentValidatorsFromDPOS(ctx)
	}

	validatorsList := ctx.Validators()
	chainID := ctx.Block().ChainID

	if len(validatorsList) == 0 {
		return nil, ErrEmptyValidatorsList
	}

	validators := make([]loom.Address, 0, len(validatorsList))
	for _, v := range validatorsList {
		if v != nil {
			address := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(v.PubKey)}
			validators = append(validators, address)
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

func addFeature(ctx contract.Context, name string, buildNumber uint64, autoEnable bool) error {
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
		AutoEnable:  autoEnable,
	}

	if err := ctx.Set(featureKey(name), &feature); err != nil {
		return err
	}

	return nil
}

func removeFeature(ctx contract.Context, name string) error {
	if name == "" {
		return ErrInvalidRequest
	}
	if ok, _ := ctx.HasPermission(addFeaturePerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}
	if found := ctx.Has(featureKey(name)); !found {
		return ErrFeatureNotFound
	}
	if ctx.FeatureEnabled(name, false) {
		return ErrFeatureAlreadyEnabled
	}
	ctx.Delete(featureKey(name))
	return nil
}

func getConfig(ctx contract.StaticContext, name string, curValidators []loom.Address) (*Config, error) {
	var config Config
	if err := ctx.Get(configKey(name), &config); err != nil {
		return nil, err
	}

	if config.Status != ConfigVoting {
		return &config, nil
	}

	// Calculate percentage of voted candidates by validators
	validatorsHashMap := map[string]bool{}
	candidateScores := map[string]int{}

	for _, v := range curValidators {
		validatorsHashMap[v.String()] = true
	}
	for _, vote := range config.Votes {
		// Only count valid validator votes
		validator := loom.UnmarshalAddressPB(vote.Validator)
		if validatorsHashMap[validator.String()] {
			candidateScores[vote.Value]++
		}
	}

	if len(curValidators) == 0 {
		return &config, nil
	}

	proposals := make([]*Proposal, 0)
	for value, voteScore := range candidateScores {
		proposal := &Proposal{
			Value:      value,
			Percentage: uint64((voteScore * 100) / len(curValidators)),
		}
		proposals = append(proposals, proposal)
	}
	config.Proposals = proposals

	return &config, nil
}

func getMostPopularProposal(proposals []*Proposal) *Proposal {
	var proposal *Proposal
	for i, p := range proposals {
		if i == 0 {
			proposal = p
			continue
		}
		if p.Percentage > proposal.Percentage {
			proposal = p
		}
	}
	return proposal
}

var Contract plugin.Contract = contract.MakePluginContract(&ChainConfig{})

func (c *ChainConfig) SetValidatorInfo(ctx contract.Context, req *SetValidatorInfoRequest) error {
	if req.BuildNumber == 0 {
		return ErrInvalidRequest
	}
	if !ctx.FeatureEnabled(loomchain.ChainCfgVersion1_2, false) {
		return ErrFeatureNotEnabled
	}
	senderAddr := ctx.Message().Sender
	validators, err := getCurrentValidators(ctx)
	if err != nil {
		return err
	}
	isValidator := false
	for _, validator := range validators {
		if validator.Compare(senderAddr) == 0 {
			isValidator = true
			break
		}
	}
	if !isValidator {
		return ErrNotAuthorized
	}

	validator := &ValidatorInfo{
		Address:     senderAddr.MarshalPB(),
		BuildNumber: req.BuildNumber,
		UpdatedAt:   uint64(ctx.Now().Unix()),
	}
	return ctx.Set(validatorInfoKey(senderAddr), validator)
}

func (c *ChainConfig) GetValidatorInfo(ctx contract.StaticContext, req *GetValidatorInfoRequest) (*GetValidatorInfoResponse, error) {
	if req.Address == nil {
		return nil, ErrInvalidRequest
	}
	address := loom.UnmarshalAddressPB(req.Address)

	var validatorInfo ValidatorInfo
	err := ctx.Get(validatorInfoKey(address), &validatorInfo)
	if err == contract.ErrNotFound {
		return &GetValidatorInfoResponse{}, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve validator info")
	}
	return &GetValidatorInfoResponse{
		Validator: &validatorInfo,
	}, nil
}

// ListValidatorsInfo returns the build number for each validators
func (c *ChainConfig) ListValidatorsInfo(ctx contract.StaticContext, req *ListValidatorsInfoRequest) (*ListValidatorsInfoResponse, error) {
	validatorRange := ctx.Range([]byte(validatorInfoPrefix))
	validators := []*ValidatorInfo{}
	for _, m := range validatorRange {
		var v ValidatorInfo
		if err := proto.Unmarshal(m.Value, &v); err != nil {
			return nil, errors.Wrapf(err, "unmarshal validators %s", string(m.Key))
		}
		validators = append(validators, &v)
	}

	return &ListValidatorsInfoResponse{
		Validators: validators,
	}, nil
}
