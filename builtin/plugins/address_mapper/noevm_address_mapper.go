// +build !evm

package address_mapper

import (
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type (
	InitRequest = amtypes.AddressMapperInitRequest
)

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

var Contract plugin.Contract = contract.MakePluginContract(&AddressMapper{})
