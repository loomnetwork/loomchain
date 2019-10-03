package loomprecompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/evm/precompiles"
)

func NewLoomPrecompileHandler(
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error),
) precompiles.EvmPrecompilerHandler {
	return &loomPrecompileHandler{
		createAddressMapperCtx: createAddressMapperCtx,
	}
}

type loomPrecompileHandler struct {
	createAddressMapperCtx func(loomchain.State) (contractpb.StaticContext, error)
}

func (h loomPrecompileHandler) AddEvmPrecompiles(_state loomchain.State) {
	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{byte(int(precompiles.MapToLoomAddress))})] =
		NewMapToLoomAddress(_state, h.createAddressMapperCtx)
}
