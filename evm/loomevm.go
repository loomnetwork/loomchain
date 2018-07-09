// +build evm

package evm

import (
	"crypto/sha256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
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

var LoomVmFactory = func(state loomchain.State) vm.VM {
	return NewLoomVm(state, nil)
}

type LoomVm struct {
	state        loomchain.State
	eventHandler loomchain.EventHandler
}

func NewLoomVm(loomState loomchain.State, eventHandler loomchain.EventHandler) vm.VM {
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
	var events []*loomchain.EventData
	if err == nil {
		events = lvm.getEvents(levm.evm.state.Logs(), caller, addr, code)
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

func (lvm LoomVm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	_, err := levm.evm.Call(caller, addr, input)
	if err == nil {
		_, err = levm.Commit()
	}

	var events []*loomchain.EventData
	if err == nil {
		events = lvm.getEvents(levm.evm.state.Logs(), caller, addr, input)
	}
	return lvm.saveEventsAndHashReceipt(caller, addr, events, err)
}

func (lvm LoomVm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	return levm.evm.StaticCall(caller, addr, input)
}

func (lvm LoomVm) GetCode(addr loom.Address) []byte {
	levm := NewLoomEvm(*lvm.state.(*loomchain.StoreState))
	return levm.GetCode(addr)
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
