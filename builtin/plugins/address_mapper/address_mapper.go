package address_mapper

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
)

type (
	AddressMapping = amtypes.AddressMapperMapping

	InitRequest          = amtypes.AddressMapperInitRequest
	AddMappingRequest    = amtypes.AddressMapperAddMappingRequest
	RemoveMappingRequest = amtypes.AddressMapperRemoveMappingRequest
	GetMappingRequest    = amtypes.AddressMapperGetMappingRequest
	GetMappingResponse   = amtypes.AddressMapperGetMappingResponse
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("invalid request")
)

func addressKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("addr"), addr.Bytes())
}

type AddressMapper struct {
}

func (am *AddressMapper) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "addressmapper",
		Version: "0.1.0",
	}, nil
}

func (am *AddressMapper) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

func (am *AddressMapper) AddMapping(ctx contract.Context, req *AddMappingRequest) error {
	if req.From == nil || req.To == nil {
		return ErrInvalidRequest
	}
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)
	if from.Compare(to) == 0 {
		return ErrInvalidRequest
	}
	// TODO: probably need to validate both the chain & local fields are set for each address too
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

func (am *AddressMapper) RemoveMapping(ctx contract.StaticContext, req *RemoveMappingRequest) error {
	// TODO
	return nil
}

func (am *AddressMapper) GetMapping(ctx contract.StaticContext, req *GetMappingRequest) (*GetMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}
	var mapping AddressMapping
	if err := ctx.Get(addressKey(loom.UnmarshalAddressPB(req.From)), &mapping); err != nil {
		return nil, err
	}
	return &GetMappingResponse{
		From: mapping.From,
		To:   mapping.To,
	}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&AddressMapper{})
