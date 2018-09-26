package registry

import (
	"errors"

	"github.com/loomnetwork/go-loom"
)

const (
	// if contract has atleast one version registered
	// this version key will exists. This help us
	// in detecting whether contract has any version
	// registered.
	SentinelVersion = "__v"

	DefaultContractVersion = ""
)

var (
	ErrAlreadyRegistered = errors.New("name is already registered")
	ErrNotFound          = errors.New("name is not registered")
	ErrInvalidVersion    = errors.New("invalid registry version")
	ErrNotImplemented    = errors.New("not implemented in this registry version")

	ErrInvalidContractVersion = errors.New("invalid contract version")
)

// Registry stores contract meta data.
// NOTE: This interface must remain backwards compatible, you may add new functions, but existing
// function signatures must remain unchanged in all released builds.
type Registry interface {
	// Register stores the given contract meta data
	Register(contractName string, contractVersion string, contractAddr, ownerAddr loom.Address) error
	// Resolve looks up the address of the contract matching the given name and optionally  version
	Resolve(contractName, contractVersion string) (loom.Address, error)
	// GetRecord looks up the meta data previously stored for the given contract
	GetRecord(contractAddr loom.Address) (*Record, error)
}
