// +build evm

package evm

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
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
	return NewLoomVm(state, nil, nil), nil
}

// LoomVm implements the loomchain/vm.VM interface using the EVM.
// TODO: rename to LoomEVM
type LoomVm struct {
	state        loomchain.State
	eventHandler loomchain.EventHandler
	createABM    AccountBalanceManagerFactoryFunc
}

func NewLoomVm(loomState loomchain.State, eventHandler loomchain.EventHandler, createABM AccountBalanceManagerFactoryFunc) vm.VM {
	return &LoomVm{
		state:        loomState,
		eventHandler: eventHandler,
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
	txHash, err := lvm.saveEventsAndHashReceipt(caller, addr, events, err)

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
	return lvm.saveEventsAndHashReceipt(caller, addr, events, err)
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

func (lvm LoomVm) saveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	sState := *lvm.state.(*loomchain.StoreState)
	ssBlock := sState.Block()
	var status int32
	if err == nil {
		status = 1
	} else {
		status = 0
	}
	txReceipt := ptypes.EvmTxReceipt{
		TransactionIndex:  sState.Block().NumTxs,
		BlockHash:         ssBlock.GetLastBlockID().Hash,
		BlockNumber:       sState.Block().Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         query.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}

	preTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return []byte{}, errMarshal
		} else {
			return []byte{}, err
		}
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)

	txReceipt.TxHash = txHash
	for _, event := range events {
		event.TxHash = txHash
		_ = lvm.eventHandler.Post(lvm.state, event)
		pEvent := ptypes.EventData(*event)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}

	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return []byte{}, errMarshal
		} else {
			return []byte{}, err
		}
	}

	receiptState := store.PrefixKVStore(utils.ReceiptPrefix, lvm.state)
	receiptState.Set(txHash, postTxReceipt)

	height := utils.BlockHeightToBytes(uint64(sState.Block().Height))
	bloomState := store.PrefixKVStore(utils.BloomPrefix, lvm.state)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(utils.TxHashPrefix, lvm.state)
	txHashState.Set(height, txReceipt.TxHash)

	return txHash, err
}

func (lvm LoomVm) getEvents(logs []*types.Log, caller, contract loom.Address, input []byte) []*loomchain.EventData {
	storeState := *lvm.state.(*loomchain.StoreState)
	var events []*loomchain.EventData

	if lvm.eventHandler == nil {
		return events
	}
	for _, log := range logs {
		var topics []string
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		eventData := &loomchain.EventData{
			Topics:          topics,
			Caller:          caller.MarshalPB(),
			Address:         contract.MarshalPB(),
			BlockHeight:     uint64(storeState.Block().Height),
			PluginName:      contract.Local.String(),
			EncodedBody:     log.Data,
			OriginalRequest: input,
		}
		events = append(events, eventData)
	}
	return events
}
