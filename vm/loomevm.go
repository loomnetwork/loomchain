// +build evm

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/go-loom"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/events"
)

var rootKey = []byte("vmroot")

var LoomEvmFactory = func(state loomchain.State) VM {
	return *NewLoomEvm(state)
}

type LoomEvm struct {
	db  ethdb.Database
	evm Evm
}

func NewLoomEvm(loomState loomchain.State) *LoomEvm {
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

func (levm LoomEvm) Commit() (common.Hash, error) {
	root, err := levm.evm.Commit()
	if err == nil {
		levm.db.Put(rootKey, root[:])
	}
	return root, err
}

var LoomVmFactory = func(state loomchain.State) VM {
	return NewLoomVm(state, loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher()))
}

type LoomVm struct {
	state        loomchain.State
	eventHandler loomchain.EventHandler
}

func NewLoomVm(loomState loomchain.State, eventHandler loomchain.EventHandler) VM {
	p := new(LoomVm)
	p.state = loomState
	p.eventHandler = eventHandler
	return p
}

func (lvm LoomVm) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	levm := NewLoomEvm(lvm.state)
	ret, addr, err := levm.evm.Create(caller, code)
	if err == nil {
		_, err = levm.Commit()
	}
	lvm.postLogs(levm.evm.state.Logs(), caller.ChainID)
	return ret, addr, err
}

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(lvm.state)
	ret, err := levm.evm.Call(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	lvm.postLogs(levm.evm.state.Logs(), caller.ChainID)
	return ret, err
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(lvm.state)
	ret, err := levm.evm.StaticCall(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	lvm.postLogs(levm.evm.state.Logs(), caller.ChainID)
	return ret, err
}

func (lvm LoomVm) postLogs(logs []*types.Log, chiainId string) error {
	for _, log := range logs {
		var topics [][]byte
		for _, topic := range log.Topics {
			topics = append(topics, topic.Bytes())
		}
		flatLog, err := proto.Marshal(&Event{
			Contract: &ltypes.Address{
				ChainId: chiainId,
				Local:   log.Address.Bytes(),
			},
			Topics: topics,
			Data:   log.Data,
		})
		if err != nil {
			return err
		}
		err = lvm.eventHandler.Post(lvm.state, flatLog)
		if err != nil {
			return err
		}
	}
	return nil
}
