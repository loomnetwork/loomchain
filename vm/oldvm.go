package vm

import (
	"github.com/loomnetwork/loom"
)

type VM interface {
	Create(caller loom.Address, code []byte) ([]byte, loom.Address, error)
	Call(caller, addr loom.Address, input []byte) ([]byte, error)
	StaticCall(caller, addr loom.Address, input []byte) ([]byte, error)
}


