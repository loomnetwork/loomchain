package contract

import (
	"github.com/loomnetwork/loom"
)

type Contract interface {
	Init(params []byte) error
	Call(state loom.State, method string, params []byte) ([]byte, error)
}
