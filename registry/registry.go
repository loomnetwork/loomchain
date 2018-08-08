package registry

import (
	"errors"

	"github.com/loomnetwork/go-loom"
)

var (
	ErrAlreadyRegistered = errors.New("name is already registered")
	ErrNotFound          = errors.New("name is not registered")
	ErrInvalidVersion    = errors.New("invalid registry version")
	ErrNotImplemented    = errors.New("not implemented in this registry version")
)

// Registry stores contract meta data.
// NOTE: This interface must remain backwards compatible, you may add new functions, but existing
// function signatures must remain unchanged in all released builds.
type Registry interface {
	// Register stores the given contract meta data
	Register(contractName string, contractAddr, ownerAddr loom.Address) error
	// Resolve looks up the address of the contract matching the given name
	Resolve(contractName string) (loom.Address, error)
	// GetRecord looks up the meta data previously stored for the given contract
	GetRecord(contractAddr loom.Address) (*Record, error)
}
