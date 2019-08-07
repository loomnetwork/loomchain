// +build evm

package evm

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/handler"
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
	loomState loomchain.State, accountBalanceManager AccountBalanceManager,
	logContext *ethdbLogContext, debug bool,
) (*LoomEvm, error) {
	p := new(LoomEvm)
	p.db = NewLoomEthdb(loomState, logContext)
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

	p.Evm = NewEvm(p.sdb, loomState, abm, debug)
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

func (levm LoomEvm) RawDump() []byte {
	d := levm.sdb.RawDump()
	output, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		panic(err)
	}
	return output
}

var LoomVmFactory = func(state loomchain.State) (vm.VM, error) {
	//TODO , debug bool, We should be able to pass in config
	debug := false
	eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())
	receiptHandlerProvider := receipts.NewReceiptHandlerProvider(
		eventHandler,
		handler.DefaultMaxReceipts,
		nil,
	)
	receiptHandler := receiptHandlerProvider.Writer()
	return NewLoomVm(state, eventHandler, receiptHandler, nil, debug), nil
}

// LoomVm implements the loomchain/vm.VM interface using the EVM.
// TODO: rename to LoomEVM
type LoomVm struct {
	state          loomchain.State
	receiptHandler loomchain.WriteReceiptHandler
	createABM      AccountBalanceManagerFactoryFunc
	debug          bool
}

func NewLoomVm(
	loomState loomchain.State,
	eventHandler loomchain.EventHandler,
	receiptHandler loomchain.WriteReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
	debug bool,
) vm.VM {
	return &LoomVm{
		state:          loomState,
		receiptHandler: receiptHandler,
		createABM:      createABM,
		debug:          debug,
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
	levm, err := NewLoomEvm(lvm.state, lvm.accountBalanceManager(false), logContext, lvm.debug)
	if err != nil {
		return nil, loom.Address{}, err
	}
	bytecode, addr, err := levm.Create(caller, code, value)
	if err == nil {
		_, err = levm.Commit()
	}

	var txHash []byte
	if lvm.receiptHandler != nil {
		var events []*ptypes.EventData
		if err == nil {
			events = lvm.receiptHandler.GetEventsFromLogs(
				levm.sdb.Logs(), lvm.state.Block().Height, caller, addr, code,
			)
		}

		var errSaveReceipt error
		txHash, errSaveReceipt = lvm.receiptHandler.CacheReceipt(lvm.state, caller, addr, events, err)
		if errSaveReceipt != nil {
			err = errors.Wrapf(err, "trouble saving receipt %v", errSaveReceipt)
		}
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
	levm, err := NewLoomEvm(lvm.state, lvm.accountBalanceManager(false), logContext, lvm.debug)
	if err != nil {
		return nil, err
	}
	_, err = levm.Call(caller, addr, input, value)
	if err == nil {
		_, err = levm.Commit()
	}

	var txHash []byte
	if lvm.receiptHandler != nil {
		var events []*ptypes.EventData
		if err == nil {
			events = lvm.receiptHandler.GetEventsFromLogs(
				levm.sdb.Logs(), lvm.state.Block().Height, caller, addr, input,
			)
		}

		var errSaveReceipt error
		txHash, errSaveReceipt = lvm.receiptHandler.CacheReceipt(lvm.state, caller, addr, events, err)
		if errSaveReceipt != nil {
			err = errors.Wrapf(err, "trouble saving receipt %v", errSaveReceipt)
		}
	}
	return txHash, err
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm, err := NewLoomEvm(lvm.state, lvm.accountBalanceManager(true), nil, lvm.debug)
	if err != nil {
		return nil, err
	}
	return levm.StaticCall(caller, addr, input)
}

func (lvm LoomVm) GetCode(addr loom.Address) ([]byte, error) {
	levm, err := NewLoomEvm(lvm.state, nil, nil, lvm.debug)
	if err != nil {
		return nil, err
	}
	return levm.GetCode(addr), nil
}
