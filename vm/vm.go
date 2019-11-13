package vm

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
)

type VM interface {
	Create(caller loom.Address, code []byte, value *loom.BigUInt) ([]byte, loom.Address, error)
	Call(caller, addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error)
	StaticCall(caller, addr loom.Address, input []byte) ([]byte, error)
	GetCode(addr loom.Address) ([]byte, error)
	GetStorageAt(addr loom.Address, hash []byte) ([]byte, error)
	GetStorageSize(addr loom.Address) (uint64, error)
}

type Factory func(loomchain.State) (VM, error)

type Manager struct {
	vms map[VMType]Factory
}

func NewManager() *Manager {
	return &Manager{
		vms: make(map[VMType]Factory),
	}
}

func (m *Manager) Register(typ VMType, fac Factory) {
	m.vms[typ] = fac
}

func (m *Manager) InitVM(typ VMType, state loomchain.State) (VM, error) {
	fac, ok := m.vms[typ]
	if !ok {
		return nil, errors.New("vm type not found")
	}

	return fac(state)
}
