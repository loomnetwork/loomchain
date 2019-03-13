package chainconfig

import (
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	AddressMapping = amtypes.AddressMapperMapping

	InitRequest               = amtypes.AddressMapperInitRequest
	AddIdentityMappingRequest = amtypes.AddressMapperAddIdentityMappingRequest
	RemoveMappingRequest      = amtypes.AddressMapperRemoveMappingRequest
	GetMappingRequest         = amtypes.AddressMapperGetMappingRequest
	GetMappingResponse        = amtypes.AddressMapperGetMappingResponse

	HasMappingRequest  = amtypes.AddressMapperHasMappingRequest
	HasMappingResponse = amtypes.AddressMapperHasMappingResponse

	ListMappingRequest  = amtypes.AddressMapperListMappingRequest
	ListMappingResponse = amtypes.AddressMapperListMappingResponse
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[ChainConfig] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[ChainConfig] invalid request")
)

func addressKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(AddressPrefix), addr.Bytes())
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

// AddIdentityMapping adds a mapping between a DAppChain account and a Mainnet account.
// The caller must provide proof of ownership of the Mainnet account.
func (f *AddressMapper) AddIdentityMapping(ctx contract.Context, req *AddIdentityMappingRequest) error {
	if req.From == nil || req.To == nil || req.Signature == nil {
		return ErrInvalidRequest
	}
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)
	if from.ChainID == "" || to.ChainID == "" {
		return ErrInvalidRequest
	}
	if from.Compare(to) == 0 {
		return ErrInvalidRequest
	}

	callerAddr := ctx.Message().Sender
	if callerAddr.Compare(from) == 0 {
		if err := verifySig(from, to, to.ChainID, req.Signature); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else if callerAddr.Compare(to) == 0 {
		if err := verifySig(from, to, from.ChainID, req.Signature); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else {
		return ErrInvalidRequest
	}

	var existingMapping AddressMapping
	if err := ctx.Get(addressKey(from), &existingMapping); err != contract.ErrNotFound {
		if err == nil {
			return ErrAlreadyRegistered
		}
		return err
	}
	if err := ctx.Get(addressKey(to), &existingMapping); err != contract.ErrNotFound {
		if err == nil {
			return ErrAlreadyRegistered
		}
		return err
	}

	err := ctx.Set(addressKey(from), &AddressMapping{
		From: req.From,
		To:   req.To,
	})
	if err != nil {
		return err
	}
	err = ctx.Set(addressKey(to), &AddressMapping{
		From: req.To,
		To:   req.From,
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *Features) ListFeatures(ctx contract.StaticContext, req *ListFeaturesRequest) (*ListMappingResponse, error) {
	mappingRange := ctx.Range([]byte(AddressPrefix))
	listMappingResponse := ListMappingResponse{
		Mappings: []*AddressMapping{},
	}

	for _, m := range mappingRange {
		var mapping AddressMapping
		if err := proto.Unmarshal(m.Value, &mapping); err != nil {
			return &ListMappingResponse{}, errors.Wrap(err, "unmarshal mapping")
		}
		listMappingResponse.Mappings = append(listMappingResponse.Mappings, &AddressMapping{
			From: mapping.From,
			To:   mapping.To,
		})
	}

	return &listMappingResponse, nil
}

func (f *Features) GetFeature(ctx contract.StaticContext, req *GetMappingRequest) (*GetMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}
	var mapping AddressMapping
	addr := loom.UnmarshalAddressPB(req.From)
	if err := ctx.Get(addressKey(addr), &mapping); err != nil {
		return nil, errors.Wrapf(err, "[Address Mapper] failed to map address %v", addr)
	}
	return &GetMappingResponse{
		From: mapping.From,
		To:   mapping.To,
	}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Features{})
