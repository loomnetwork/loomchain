package evm

import (
	"fmt"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain"
)

func NewMapToLoomAddress(
	_state loomchain.State,
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error),
) *mapToLoomAccount {
	return &mapToLoomAccount{
		state:                  _state,
		createAddressMapperCtx: createAddressMapperCtx,
	}
}

type mapToLoomAccount struct {
	state                  loomchain.State
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error)
}

func (ma mapToLoomAccount) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (ma mapToLoomAccount) Run(input []byte) ([]byte, error) {
	strI := string(input)
	_ = strI
	fmt.Printf("in mapToLoomAccount input hex %x str string %s bytes %v\n", input, input, input)

	addr := loom.Address{
		ChainID: string(input[20:]),
		Local:   input[:20],
	}
	return addr.Local, nil
	/*
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
			return nil, fmt.Errorf("mapped account %s has wrong chain ID", addr.String())
		}

		return mappedAddr.Local, nil
	*/
}
