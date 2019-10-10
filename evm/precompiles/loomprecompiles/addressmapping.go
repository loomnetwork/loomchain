package loomprecompiles

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
)

func NewMapToLoomAddress(
	_state loomchain.State,
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error),
) *mapToLoomAddress {
	return &mapToLoomAddress{
		state:                  _state,
		createAddressMapperCtx: createAddressMapperCtx,
	}
}

type mapToLoomAddress struct {
	state                  loomchain.State
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error)
}

func (ma mapToLoomAddress) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (ma mapToLoomAddress) Run(input []byte) ([]byte, error) {
	addr := loom.Address{
		ChainID: string(input[20:]),
		Local:   input[:20],
	}

	ctx, err := ma.createAddressMapperCtx(ma.state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Address Mapper context")
	}
	am := &address_mapper.AddressMapper{}

	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: addr.MarshalPB(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to map account %s", addr.String())
	}

	mappedAddr := loom.UnmarshalAddressPB(resp.To)
	if mappedAddr.ChainID != ma.state.Block().ChainID {
		return nil, errors.Errorf("mapped account %s has wrong chain ID", addr.String())
	}

	return mappedAddr.Local, nil
}

func NewMapToAddress(
	_state loomchain.State,
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error),
) *mapToAddress {
	return &mapToAddress{
		state:                  _state,
		createAddressMapperCtx: createAddressMapperCtx,
	}
}

type mapToAddress struct {
	state                  loomchain.State
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error)
}

func (ma mapToAddress) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (ma mapToAddress) Run(input []byte) ([]byte, error) {
	strI := string(input)
	_ = strI

	if len(input) < 23 || uint(len(input)) < 22+uint(input[20]) {
		return nil, errors.Errorf("mapBetweenAccounts input too short %x", input)
	}
	local := input[:20]
	chainFromLen := uint(input[20])
	chainFrom := string(input[21 : 21+chainFromLen])
	chainTo := string(input[21+chainFromLen:])
	addr := loom.Address{
		ChainID: chainFrom,
		Local:   local,
	}

	ctx, err := ma.createAddressMapperCtx(ma.state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Address Mapper context")
	}
	am := &address_mapper.AddressMapper{}

	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: addr.MarshalPB(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to map account %s", addr.String())
	}

	mappedAddr := loom.UnmarshalAddressPB(resp.To)
	if mappedAddr.ChainID != chainTo {
		return nil, errors.Errorf(
			"mapped account %s has wrong chain ID, looking for %s found %s",
			addr.String(), addr.ChainID, chainTo,
		)
	}

	return mappedAddr.Local, nil
}
