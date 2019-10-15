package precompiles

import (
	"github.com/loomnetwork/loomchain"
)

const (
	LoomPrecompilesStartIndex = 0x20
	MapToAddress              = iota + LoomPrecompilesStartIndex
)

type EvmPrecompilerHandler interface {
	AddEvmPrecompiles(loomchain.State)
}
