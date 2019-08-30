package genesis

import (
	"encoding/json"

	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	lvm "github.com/loomnetwork/go-loom/vm"
)

// This stuff is here to avoid cyclic dependencies between the config package and various other
// packages.

type ContractConfig struct {
	VMTypeName string          `json:"vm"`
	Format     string          `json:"format,omitempty"`
	Name       string          `json:"name,omitempty"`
	Location   string          `json:"location"`
	Init       json.RawMessage `json:"init"`
}

func (c ContractConfig) VMType() lvm.VMType {
	return lvm.VMType(lvm.VMType_value[c.VMTypeName])
}

type Genesis struct {
	Contracts []ContractConfig `json:"contracts"`
	Config    cctypes.Config   `json:"config"`
}
