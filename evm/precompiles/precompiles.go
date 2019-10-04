package precompiles

import (
	"github.com/loomnetwork/loomchain"
)

const (
	LoomPrecompilesStartIndex = 0x20
	MapToLoomAddress          = iota + LoomPrecompilesStartIndex
	MapToAddress
)

type EvmPrecompilerHandler interface {
	AddEvmPrecompiles(loomchain.State)
}
