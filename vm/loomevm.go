// +build evm

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"

	"crypto/sha256"

	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

var rootKey = []byte("vmroot")

var LoomEvmFactory = func(state loomchain.State) VM {
	return *NewMockLoomEvm()
}

type LoomEvm struct {
	db  ethdb.Database
	evm Evm
}

func NewLoomEvm(loomState loomchain.StoreState) *LoomEvm {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(&loomState)
	oldRoot, _ := p.db.Get(rootKey)
	_state, _ := state.New(common.BytesToHash(oldRoot), state.NewDatabase(p.db))
	p.evm = *NewEvm(*_state, loomState)
	return p
}

func NewMockLoomEvm() *LoomEvm {
	p := new(LoomEvm)
	p.evm = *NewMockEvm()
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

func (levm LoomEvm) GetCode(addr loom.Address) []byte {
	return levm.evm.GetCode(addr)
}

var LoomVmFactory = func(state loomchain.State) VM {
	return NewLoomVm(state, nil)
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
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	bytecode, addr, err := levm.evm.Create(caller, code)
	if err == nil {
		_, err = levm.Commit()
	}
	var events []*ptypes.EventData
	if err == nil {
		events, err = lvm.postEvents(levm.evm.state.Logs(), caller, addr, code)
	}
	txHash, err := lvm.saveReceipt(addr, events, err)
	response, errMarshal := proto.Marshal(&DeployResponseData{
		TxHash:   txHash,
		Bytecode: bytecode,
	})
	if errMarshal != nil {
		if err == nil {
			return []byte{}, addr, errMarshal
		} else {
			return []byte{}, addr, err
		}
	}
	return response, addr, err
}

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	_, err := levm.evm.Call(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	var events []*ptypes.EventData
	if err == nil {
		events, err = lvm.postEvents(levm.evm.state.Logs(), caller, addr, input)
	}
	return lvm.saveReceipt(addr, events, err)
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	ret, err := levm.evm.StaticCall(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}
	return ret, err
}

func (lvm LoomVm) GetCode(addr loom.Address) []byte {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	return levm.GetCode(addr)
}

func (lvm LoomVm) saveReceipt(addr loom.Address, events []*ptypes.EventData, err error) ([]byte, error) {
	storeState := *lvm.state.(*loomchain.StoreState)
	ssBlock := storeState.Block()
	var status int32
	if err == nil {
		status = 1
	} else {
		status = 0
	}
	txReceipt, errMarshal := proto.Marshal(&ptypes.EvmTxReceipt{
		TransactionIndex:  storeState.Block().NumTxs,
		BlockHash:         ssBlock.GetLastBlockID().Hash,
		BlockNumber:       storeState.Block().Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		Logs:              events,
		LogsBloom:         []byte{},
		Status:            status,
	})
	if errMarshal != nil {
		if err == nil {
			return []byte{}, errMarshal
		} else {
			return []byte{}, err
		}
	}
	h := sha256.New()
	h.Write(txReceipt)
	txHash := h.Sum(nil)
	receiptState := store.PrefixKVStore(ReceiptPrefix, lvm.state)
	receiptState.Set(txHash, txReceipt)
	return txHash, err
}

func (lvm LoomVm) postEvents(logs []*types.Log, caller, contract loom.Address, input []byte) ([]*ptypes.EventData, error) {
	var events []*ptypes.EventData
	if lvm.eventHandler == nil {
		return events, nil
	}
	for _, log := range logs {
		var topics []string
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		eventData := &ptypes.EventData{
			Topics:          topics,
			Caller:          caller.MarshalPB(),
			Address:         contract.MarshalPB(),
			PluginName:      contract.String(),
			EncodedBody:     log.Data,
			OriginalRequest: input,
		}
		err := lvm.eventHandler.Post(lvm.state, eventData)
		if err != nil {
			return []*ptypes.EventData{}, err
		}
		events = append(events, eventData)
	}
	return events, nil
}
