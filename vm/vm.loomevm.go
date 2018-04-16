package vm

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

var rootKey = []byte("vmroot")

type LoomEvm struct {
	db ethdb.Database
	evm Evm
}

func NewLoomEvm(loomState loom.State) *LoomEvm {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(loomState)
	oldRoot := loomState.Get(rootKey)
	_state, _ := state.New(common.BytesToHash(oldRoot), state.NewDatabase(p.db))
	p.evm = *NewEvmFrom(_state)
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