package chainconfig

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest          = chainconfig.ChainConfigInitRequest
	ListFeaturesRequest  = chainconfig.ListFeaturesRequest
	ListFeaturesResponse = chainconfig.ListFeaturesResponse

	GetFeatureRequest = chainconfig.GetFeatureRequest
	Feature           = chainconfig.Feature
	Config            = chainconfig.Config

	UpdateFeatureRequest = chainconfig.UpdateFeatureRequest
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[ChainConfig] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[ChainConfig] invalid request")
)

func configKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(configPrefix), addr.Bytes())
}

type ChainConfig struct {
}

func (c *ChainConfig) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "chainconfig",
		Version: "0.1.0",
	}, nil
}

func (c *ChainConfig) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

//TODO: require manual sign by all the validators, its fairly safe, for now we allow one way movement
//worst thing would be a consensus issue

//TODO: first pass only has features, which are a subset of configs
//that are only boolean

// SetFeature
func (c *ChainConfig) UpdateFeature(ctx contract.Context, req *UpdateFeatureRequest) error {
	return nil
}

func (c *ChainConfig) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListFeaturesResponse, error) {
	listFeatureResponse := listFeatureResponse{
		Feature: []*Feature{},
	}

	return &listMappingResponse, nil
}

func (c *ChainConfig) GetFeature(ctx contract.StaticContext, req *GetFeatureRequest) (*Feature, error) {
	return &GetMappingResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Features{})
