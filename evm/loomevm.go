// +build evm

package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	vmPrefix = []byte("vm")
	rootKey  = []byte("vmroot")
)

type StateDB interface {
	ethvm.StateDB
	Database() state.Database
	Logs() []*types.Log
	Commit(bool) (common.Hash, error)
}

// TODO: this doesn't need to be exported, rename to loomEvmWithState
type LoomEvm struct {
	*Evm
	db  ethdb.Database
	sdb StateDB
}

// TODO: this doesn't need to be exported, rename to newLoomEvmWithState
func NewLoomEvm(loomState loomchain.StoreState, accountBalanceManager AccountBalanceManager) (*LoomEvm, error) {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(&loomState)
	oldRoot, err := p.db.Get(rootKey)
	if err != nil {
		return nil, err
	}

	var abm *evmAccountBalanceManager
	if accountBalanceManager != nil {
		abm = newEVMAccountBalanceManager(accountBalanceManager, loomState.Block().ChainID)
		p.sdb, err = newLoomStateDB(abm, common.BytesToHash(oldRoot), state.NewDatabase(p.db))
	} else {
		p.sdb, err = state.New(common.BytesToHash(oldRoot), state.NewDatabase(p.db))
	}
	if err != nil {
		return nil, err
	}

	p.Evm = NewEvm(p.sdb, loomState, abm)
	return p, nil
}

func (levm LoomEvm) Commit() (common.Hash, error) {
	root, err := levm.sdb.Commit(true)
	if err != nil {
		return root, err
	}
	if err := levm.sdb.Database().TrieDB().Commit(root, false); err != nil {
		return root, err
	}
	if err := levm.db.Put(rootKey, root[:]); err != nil {
		return root, err
	}
	return root, err
}

var LoomVmFactory = func(state loomchain.State) (vm.VM, error) {
	return NewLoomVm(state, nil,  nil), nil
}

// LoomVm implements the loomchain/vm.VM interface using the EVM.
// TODO: rename to LoomEVM
type LoomVm struct {
	state           loomchain.State
	receiptCache    *loomchain.WriteReceiptCache
	createABM       AccountBalanceManagerFactoryFunc
}

func NewLoomVm(
		loomState       loomchain.State,
		receiptCache    *loomchain.WriteReceiptCache,
		createABM       AccountBalanceManagerFactoryFunc,
	) vm.VM {
	return &LoomVm{
		state:        loomState,
		receiptCache: receiptCache,
		createABM:    createABM,
	}
}

func (lvm LoomVm) accountBalanceManager(readOnly bool) AccountBalanceManager {
	if lvm.createABM == nil {
		return nil
	}
	return lvm.createABM(readOnly)
}

func (lvm LoomVm) Create(caller loom.Address, code []byte, value *loom.BigUInt) ([]byte, loom.Address, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(false))
	if err != nil {
		return nil, loom.Address{}, err
	}
	bytecode, addr, err := levm.Create(caller, code, value)
	if err == nil {
		_, err = levm.Commit()
	}
	var events []*loomchain.EventData
	if err == nil {
		events = lvm.getEvents(levm.sdb.Logs(), caller, addr, code)
	}
	if lvm.receiptCache == nil {
		return []byte{}, addr, err
	}
	txHash, err := (*lvm.receiptCache).SaveEventsAndHashReceipt(lvm.state, caller, addr, events, err)

	response, errMarshal := proto.Marshal(&vm.DeployResponseData{
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

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(false))
	if err != nil {
		return nil, err
	}
	_, err = levm.Call(caller, addr, input, value)
	if err == nil {
		_, err = levm.Commit()
	}

	var events []*loomchain.EventData
	if err == nil {
		events = lvm.getEvents(levm.sdb.Logs(), caller, addr, input)
	}
	if lvm.receiptCache == nil {
		return []byte{}, err
	} else {
		return (*lvm.receiptCache).SaveEventsAndHashReceipt(lvm.state, caller, addr, events, err)
	}
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(true))
	if err != nil {
		return nil, err
	}
	return levm.StaticCall(caller, addr, input)
}

func (lvm LoomVm) GetCode(addr loom.Address) ([]byte, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), nil)
	if err != nil {
		return nil, err
	}
	return levm.GetCode(addr), nil
}

func (lvm LoomVm) getEvents(logs []*types.Log, caller, contract loom.Address, input []byte) []*loomchain.EventData {
	storeState := *lvm.state.(*loomchain.StoreState)
	var events []*loomchain.EventData

	for _, log := range logs {
		var topics []string
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		eventData := &loomchain.EventData{
			Topics:          topics,
			Caller:          caller.MarshalPB(),
			Address:         loom.Address{
									ChainID: caller.ChainID,
									Local:   log.Address.Bytes(),
							 }.MarshalPB(),
			BlockHeight:     uint64(storeState.Block().Height),
			PluginName:      contract.Local.String(),
			EncodedBody:     log.Data,
			OriginalRequest: input,
		}
		events = append(events, eventData)
	}
	return events
}
