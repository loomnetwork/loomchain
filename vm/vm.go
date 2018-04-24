package vm

import (
	"errors"

	"github.com/loomnetwork/loom"
	lp "github.com/loomnetwork/loom-plugin"
)

type VM interface {
	Create(caller lp.Address, code []byte) ([]byte, lp.Address, error)
	Call(caller, addr lp.Address, input []byte) ([]byte, error)
	StaticCall(caller, addr lp.Address, input []byte) ([]byte, error)
}

type Factory func(loom.State) VM

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

func (m *Manager) InitVM(typ VMType, state loom.State) (VM, error) {
	fac, ok := m.vms[typ]
	if !ok {
		return nil, errors.New("vm type not found")
	}

	return fac(state), nil
}
