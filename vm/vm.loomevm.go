// +build evm

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/loomnetwork/loom"
)

var rootKey = []byte("vmroot")

var LoomEvmFactory = func(state loom.State) VM {
	return *NewLoomEvm(state)
}

type LoomEvm struct {
	db  ethdb.Database
	evm Evm
}

func NewLoomEvm(loomState loom.State) *LoomEvm {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(loomState)
	oldRoot, _ := p.db.Get(rootKey)
	_state, _ := state.New(common.BytesToHash(oldRoot), state.NewDatabase(p.db))
	p.evm = *NewEvmFrom(*_state)
	return p
}

func (levm LoomEvm) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	return levm.evm.Create(caller, code)
}

func (levm LoomEvm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	return levm.evm.Call(caller, addr, input)
}

func (levm LoomEvm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	return levm.evm.StaticCall(caller, addr, input)
}

func(levm LoomEvm) Commit() (common.Hash, error)  {
	root, err := levm.evm.Commit()
	if (err == nil) {
		levm.db.Put(rootKey,  root[:])
	}
	return root, err
}

var LoomVmFactory = func(state loom.State) VM {
	return *NewLoomVm(state)
}

type LoomVm struct {
	state loom.State
}

func NewLoomVm(loomState loom.State) *LoomVm {
	p := new(LoomVm)
	p.state = loomState
	return p
}

func (lvm LoomVm) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	levm := NewLoomEvm(lvm.state)
	ret, addr, err := levm.evm.Create(caller, code)
	if err == nil {
		_, err = levm.Commit()
	}
	return ret, addr, err
}

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(lvm.state)
	ret, err := levm.evm.Call(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	return ret, err
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(lvm.state)
	ret, err :=  levm.evm.StaticCall(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	return ret, err
}