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
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	vmPrefix   = []byte("vm")
	rootKey    = []byte("vmroot")
	evmRootKey = []byte("evmroot")
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
	db        ethdb.Database
	sdb       StateDB
	loomState loomchain.State
}

// TODO: this doesn't need to be exported, rename to newLoomEvmWithState
func NewLoomEvm(
	loomState loomchain.State,
	evmDB dbm.DB,
	accountBalanceManager AccountBalanceManager,
	logContext *MultiWriterDBLogContext,
	debug bool,
) (*LoomEvm, error) {
	p := new(LoomEvm)

	// If EvmDBFeature is enabled, read from and write to evm.db
	// otherwise write to both evm.db and app.db but read from app.db
	var config MultiWriterDBConfig
	var loomEthDB *LoomEthDB
	if loomState.FeatureEnabled(loomchain.EvmDBFeature, true) {
		config = MultiWriterDBConfig{
			Read:  EVM_DB,
			Write: EVM_DB,
		}
	} else {
		loomEthDB = NewLoomEthDB(loomState, MultiWriterDBLogContextToEthDbLogContext(logContext))
		config = MultiWriterDBConfig{
			Read:  LOOM_ETH_DB,
			Write: BOTH_DB,
		}
	}

	p.db = NewMultiWriterDB(evmDB, loomEthDB, logContext, config)

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
	// Save root of EVM Patricia tree in IAVL tree
	// levm.loomState.Set(evmRootKey, root[:])

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

// LoomVm implements the loomchain/vm.VM interface using the EVM.
// TODO: rename to LoomEVM
type LoomVm struct {
	state          loomchain.State
	evmDB          dbm.DB
	receiptHandler loomchain.WriteReceiptHandler
	createABM      AccountBalanceManagerFactoryFunc
	debug          bool
}

func NewLoomVm(
	loomState loomchain.State,
	evmDB dbm.DB,
	eventHandler loomchain.EventHandler,
	receiptHandler loomchain.WriteReceiptHandler,
	createABM AccountBalanceManagerFactoryFunc,
	debug bool,
) vm.VM {
	return &LoomVm{
		state:          loomState,
		evmDB:          evmDB,
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
	logContext := &MultiWriterDBLogContext{
		BlockHeight:  lvm.state.Block().Height,
		ContractAddr: loom.Address{},
		CallerAddr:   caller,
	}
	levm, err := NewLoomEvm(lvm.state, lvm.evmDB, lvm.accountBalanceManager(false), logContext, lvm.debug)
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
	logContext := &MultiWriterDBLogContext{
		BlockHeight:  lvm.state.Block().Height,
		ContractAddr: addr,
		CallerAddr:   caller,
	}
	levm, err := NewLoomEvm(lvm.state, lvm.evmDB, lvm.accountBalanceManager(false), logContext, lvm.debug)
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
	levm, err := NewLoomEvm(lvm.state, lvm.evmDB, lvm.accountBalanceManager(true), nil, lvm.debug)
	if err != nil {
		return nil, err
	}
	return levm.StaticCall(caller, addr, input)
}

func (lvm LoomVm) GetCode(addr loom.Address) ([]byte, error) {
	levm, err := NewLoomEvm(lvm.state, lvm.evmDB, nil, nil, lvm.debug)
	if err != nil {
		return nil, err
	}
	return levm.GetCode(addr), nil
}

func MultiWriterDBLogContextToEthDbLogContext(multiWriterLogContext *MultiWriterDBLogContext) *ethdbLogContext {
	var logContext *ethdbLogContext
	if multiWriterLogContext != nil {
		logContext = &ethdbLogContext{
			blockHeight:  multiWriterLogContext.BlockHeight,
			contractAddr: multiWriterLogContext.ContractAddr,
			callerAddr:   multiWriterLogContext.ContractAddr,
		}
	}
	return logContext
}
