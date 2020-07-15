package chainconfig

import (
	"strings"

	"github.com/loomnetwork/go-loom"
	goloom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/client"
	"github.com/pkg/errors"
)

type (
	ListFeaturesRequest      = cctypes.ListFeaturesRequest
	ListFeaturesResponse     = cctypes.ListFeaturesResponse
	Feature                  = cctypes.Feature
	EnableFeatureRequest     = cctypes.EnableFeatureRequest
	EnableFeatureResponse    = cctypes.EnableFeatureResponse
	SetValidatorInfo         = cctypes.SetValidatorInfoRequest
	GetValidatorInfoRequest  = cctypes.GetValidatorInfoRequest
	GetValidatorInfoResponse = cctypes.GetValidatorInfoResponse
)

const (
	// FeaturePending status indicates that a feature hasn't been enabled by the majority of validators yet.
	FeaturePending = cctypes.Feature_PENDING
	// FeatureWaiting status indicates a feature that has been enabled by the majority of validators, but
	// hasn't been activated yet because not enough blocks confirmations have occurred yet.
	FeatureWaiting = cctypes.Feature_WAITING
	// FeatureEnabled status indicates a feature that has been enabled by the majority of validators, and
	// has been activated on the chain.
	FeatureEnabled = cctypes.Feature_ENABLED
	// FeatureDisabled is not currently used.
	FeatureDisabled = cctypes.Feature_DISABLED
)

// ChainConfigClient is used to enable pending features in the ChainConfig contract.
type ChainConfigClient struct {
	Address  goloom.Address
	contract *client.Contract
	caller   goloom.Address
	logger   *goloom.Logger
	signer   auth.Signer
}

// NewChainConfigClient returns a ChainConfigClient instance
func NewChainConfigClient(
	loomClient *client.DAppChainRPCClient,
	caller goloom.Address,
	signer auth.Signer,
	logger *goloom.Logger,
) (*ChainConfigClient, error) {
	chainConfigAddr, err := loomClient.Resolve("chainconfig")
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve ChainConfig contract address")
	}
	return &ChainConfigClient{
		Address:  chainConfigAddr,
		contract: client.NewContract(loomClient, chainConfigAddr.Local),
		caller:   caller,
		signer:   signer,
		logger:   logger,
	}, nil
}

// VoteToEnablePendingFeatures is called periodically by ChainConfigRoutine
// to enable pending feature if it's supported in this current build
func (cc *ChainConfigClient) VoteToEnablePendingFeatures(buildNumber uint64) error {
	var resp ListFeaturesResponse
	if _, err := cc.contract.StaticCall(
		"ListFeatures",
		&ListFeaturesRequest{},
		cc.caller,
		&resp,
	); err != nil {
		cc.logger.Error("Failed to retrieve features from ChainConfig contract", "err", err)
		return err
	}

	features := resp.Features
	featureNames := make([]string, 0)
	for _, feature := range features {
		if feature.Status == FeaturePending &&
			feature.BuildNumber <= buildNumber &&
			feature.AutoEnable &&
			!cc.hasVoted(feature) {
			featureNames = append(featureNames, feature.Name)
		}
	}

	if len(featureNames) > 0 {
		var resp EnableFeatureResponse
		if _, err := cc.contract.Call(
			"EnableFeature",
			&EnableFeatureRequest{Names: featureNames},
			cc.signer,
			&resp,
		); err != nil {
			cc.logger.Error(
				"Encountered an error while trying to auto-enable features",
				"features", strings.Join(featureNames, ","), "err", err,
			)
			return err
		}
		cc.logger.Info("Auto-enabled features", "features", strings.Join(featureNames, ","))
	}
	return nil
}

// Check if this validator has already voted to enable this feature
func (cc *ChainConfigClient) hasVoted(feature *Feature) bool {
	for _, v := range feature.Validators {
		validator := loom.UnmarshalAddressPB(v)
		if validator.Compare(cc.caller) == 0 {
			return true
		}
	}
	return false
}

func (cc *ChainConfigClient) SetBuildNumber(buildNumber uint64) error {
	if _, err := cc.contract.Call(
		"SetValidatorInfo",
		&SetValidatorInfo{BuildNumber: buildNumber},
		cc.signer,
		nil,
	); err != nil {
		cc.logger.Error("Failed to set build number in ChainConfig contract", "err", err)
		return err
	}
	return nil
}

func (cc *ChainConfigClient) GetValidatorInfo() (*GetValidatorInfoResponse, error) {
	var resp GetValidatorInfoResponse
	if _, err := cc.contract.StaticCall(
		"GetValidatorInfo",
		&GetValidatorInfoRequest{Address: cc.caller.MarshalPB()},
		cc.caller,
		&resp,
	); err != nil {
		cc.logger.Error("Failed to get validator information in ChainConfig contract", "err", err)
		return nil, err
	}
	return &resp, nil
}
