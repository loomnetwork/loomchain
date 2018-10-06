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
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/receipts"
	rfactory "github.com/loomnetwork/loomchain/receipts/factory"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
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

type ethdbLogContext struct {
	blockHeight  int64
	contractAddr loom.Address
	callerAddr   loom.Address
}

// TODO: this doesn't need to be exported, rename to loomEvmWithState
type LoomEvm struct {
	*Evm
	db  ethdb.Database
	sdb StateDB
}

// TODO: this doesn't need to be exported, rename to newLoomEvmWithState
func NewLoomEvm(
	loomState loomchain.StoreState, accountBalanceManager AccountBalanceManager,
	logContext *ethdbLogContext,
) (*LoomEvm, error) {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(&loomState, logContext)
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
	eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())
	factory, err := rfactory.NewReceiptHandlerFactory(rfactory.ReceiptHandlerChain, eventHandler)
	if err != nil {
		return nil, errors.Wrap(err, "making receipt factory")
	}
	return NewLoomVm(state, nil, factory, nil), nil
}

// LoomVm implements the loomchain/vm.VM interface using the EVM.
// TODO: rename to LoomEVM
type LoomVm struct {
	state          loomchain.State
	receiptHandler receipts.ReceiptHandler
	createABM      AccountBalanceManagerFactoryFunc
}

func NewLoomVm(
	loomState loomchain.State,
	eventHandler loomchain.EventHandler,

	receiptHandler receipts.ReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
) vm.VM {
	return &LoomVm{
		state:          loomState,
		receiptHandler: receiptHandler,
		createABM:      createABM,
	}
}

func (lvm LoomVm) accountBalanceManager(readOnly bool) AccountBalanceManager {
	if lvm.createABM == nil {
		return nil
	}
	return lvm.createABM(readOnly)
}

func (lvm LoomVm) Create(caller loom.Address, code []byte, value *loom.BigUInt) ([]byte, loom.Address, error) {
	logContext := &ethdbLogContext{
		blockHeight:  lvm.state.Block().Height,
		contractAddr: loom.Address{},
		callerAddr:   caller,
	}
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(false), logContext)
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
	if lvm.receiptHandler == nil {
		return []byte{}, addr, err
	}
	txHash, errSaveReceipt := lvm.receiptHandler.SaveEventsAndHashReceipt(lvm.state, caller, addr, events, err)
	if errSaveReceipt != nil {
		err = errors.Wrapf(err, "trouble saving receipt %v", errSaveReceipt)
	}

	response, errMarshal := proto.Marshal(&vm.DeployResponseData{
		TxHash:   txHash,
		Bytecode: bytecode,
	})
	if errMarshal != nil {
		if err == nil {
			return []byte{}, addr, errMarshal
		} else {
			return []byte{}, addr, errors.Wrapf(err, "error marshaling %v", errMarshal)
		}
	}
	return response, addr, err
}

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	logContext := &ethdbLogContext{
		blockHeight:  lvm.state.Block().Height,
		contractAddr: addr,
		callerAddr:   caller,
	}
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(false), logContext)
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
	if lvm.receiptHandler == nil {
		return []byte{}, err
	} else {
		data, errSaveReceipt := lvm.receiptHandler.SaveEventsAndHashReceipt(lvm.state, caller, addr, events, err)
		if errSaveReceipt != nil {
			if err == nil {
				return data, errSaveReceipt
			}
			err = errors.Wrapf(err, "trouble saving receipt %v", errSaveReceipt)
		}
		return data, err
	}
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), lvm.accountBalanceManager(true), nil)
	if err != nil {
		return nil, err
	}
	return levm.StaticCall(caller, addr, input)
}

func (lvm LoomVm) GetCode(addr loom.Address) ([]byte, error) {
	levm, err := NewLoomEvm(*lvm.state.(*loomchain.StoreState), nil, nil)
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
			Topics: topics,
			Caller: caller.MarshalPB(),
			Address: loom.Address{
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
