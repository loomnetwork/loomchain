// +build evm

package plugin

import (
	"bytes"
	"context"
	"io/ioutil"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gtypes "github.com/ethereum/go-ethereum/core/types"
	proto "github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	loom_plugin "github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/testdata"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/events"
	levm "github.com/loomnetwork/loomchain/evm"
	rcommon "github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/handler"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	vmAddr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	vmAddr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
)

// Implements loomchain.EventHandler interface
type fakeEventHandler struct {
}

func (eh *fakeEventHandler) Post(height uint64, e *ptypes.EventData) error {
	return nil
}

func (eh *fakeEventHandler) EmitBlockTx(_ uint64, _ time.Time) error {
	return nil
}

func (eh *fakeEventHandler) SubscriptionSet() *loomchain.SubscriptionSet {
	return nil
}

func (eh *fakeEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return nil
}

func (eh *fakeEventHandler) LegacyEthSubscriptionSet() *subs.LegacyEthSubscriptionSet {
	return nil
}

type VMTestContract struct {
	t                          *testing.T
	Name                       string
	ExpectedCaller             loom.Address
	NextContractAddr           loom.Address
	NextContractABI            *abi.ABI
	NextContractExpectedCaller loom.Address
}

func (c *VMTestContract) reset() {
	c.ExpectedCaller = loom.Address{}
	c.NextContractAddr = loom.Address{}
	c.NextContractABI = nil
	c.NextContractExpectedCaller = loom.Address{}
}

func (c *VMTestContract) Meta() (loom_plugin.Meta, error) {
	return loom_plugin.Meta{
		Name:    c.Name,
		Version: "0.0.1",
	}, nil
}

func (c *VMTestContract) Init(ctx contract.Context, req *loom_plugin.Request) error {
	return nil
}

func (c *VMTestContract) CheckTxCaller(ctx contract.Context, args *testdata.CallArgs) error {
	require.Equal(c.t, c.ExpectedCaller, ctx.Message().Sender)
	if !c.NextContractAddr.IsEmpty() {
		if c.NextContractABI != nil {
			expectedCallerAddr := common.BytesToAddress(c.NextContractExpectedCaller.Local)
			require.NoError(c.t, callEVMContractMethod(ctx, c.NextContractAddr, c.NextContractABI, "setExpectedCaller", expectedCallerAddr))
			require.NoError(c.t, callEVMContractMethod(ctx, c.NextContractAddr, c.NextContractABI, "checkTxCaller"))
		} else {
			require.NoError(c.t, contract.CallMethod(ctx, c.NextContractAddr, "CheckTxCaller", args, nil))
		}
	}
	return nil
}

func (c *VMTestContract) CheckQueryCaller(ctx contract.StaticContext, args *testdata.StaticCallArgs) (*testdata.StaticCallResult, error) {
	require.Equal(c.t, c.ExpectedCaller, ctx.Message().Sender)
	if !c.NextContractAddr.IsEmpty() {
		if c.NextContractABI != nil {
			expectedCallerAddr := common.BytesToAddress(c.NextContractExpectedCaller.Local)
			var result common.Address
			require.NoError(c.t, staticCallEVMContractMethod(ctx, c.NextContractAddr, c.NextContractABI, "checkQueryCaller", &result))
			require.Equal(c.t, expectedCallerAddr, result)
		} else {
			require.NoError(c.t, contract.StaticCallMethod(ctx, c.NextContractAddr, "CheckQueryCaller", args, nil))
		}
	}
	return &testdata.StaticCallResult{}, nil
}

func TestPluginVMContractContextCaller(t *testing.T) {

	fc1 := &VMTestContract{t: t, Name: "fakecontract1"}
	fc2 := &VMTestContract{t: t, Name: "fakecontract2"}
	fc3 := &VMTestContract{t: t, Name: "fakecontract3"}
	loader := NewStaticLoader(
		contract.MakePluginContract(fc1),
		contract.MakePluginContract(fc2),
		contract.MakePluginContract(fc3),
	)
	block := abci.Header{
		ChainID: "chain",
		Height:  int64(34),
		Time:    time.Unix(123456789, 0),
	}
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), block, nil, nil)
	createRegistry, err := registry.NewRegistryFactory(registry.LatestRegistryVersion)
	require.NoError(t, err)

	vm := NewPluginVM(loader, state, createRegistry(state), &fakeEventHandler{}, nil, nil, nil, nil)
	evm := levm.NewLoomVm(state, nil, nil, nil, false)

	// Deploy contracts
	owner := loom.RootAddress("chain")
	goContractAddr1, err := deployGoContract(vm, "fakecontract1:0.0.1", 0, owner)
	require.NoError(t, err)
	goContractAddr2, err := deployGoContract(vm, "fakecontract2:0.0.1", 1, owner)
	require.NoError(t, err)
	goContractAddr3, err := deployGoContract(vm, "fakecontract3:0.0.1", 2, owner)
	require.NoError(t, err)
	evmContractAddr, evmContractABI, err := deployEVMContract(evm, "VMTestContract", owner)
	require.NoError(t, err)

	// Go contract -> Go contract
	fc1.ExpectedCaller = vmAddr1
	fc1.NextContractAddr = goContractAddr2
	fc2.ExpectedCaller = goContractAddr1
	require.NoError(t, callGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckTxCaller", &testdata.CallArgs{}))
	require.NoError(t, staticCallGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckQueryCaller", &testdata.StaticCallArgs{}))

	fc1.reset()
	fc2.reset()

	// Go contract -> EVM contract
	fc1.ExpectedCaller = vmAddr1
	fc1.NextContractAddr = evmContractAddr
	fc1.NextContractABI = evmContractABI
	fc1.NextContractExpectedCaller = goContractAddr1
	require.NoError(t, callGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckTxCaller", &testdata.CallArgs{}))
	require.NoError(t, staticCallGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckQueryCaller", &testdata.StaticCallArgs{}))

	fc1.reset()
	fc2.reset()
	fc3.reset()

	// Go contract -> Go contract -> Go contract
	fc1.ExpectedCaller = vmAddr1
	fc1.NextContractAddr = goContractAddr2
	fc1.NextContractABI = nil
	fc2.ExpectedCaller = goContractAddr1
	fc2.NextContractAddr = goContractAddr3
	fc3.ExpectedCaller = goContractAddr2
	require.NoError(t, callGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckTxCaller", &testdata.CallArgs{}))
	require.NoError(t, staticCallGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckQueryCaller", &testdata.StaticCallArgs{}))

	fc1.reset()
	fc2.reset()

	// Go contract -> Go contract -> EVM contract
	fc1.ExpectedCaller = vmAddr1
	fc1.NextContractAddr = goContractAddr2
	fc2.ExpectedCaller = goContractAddr1
	fc2.NextContractAddr = evmContractAddr
	fc2.NextContractABI = evmContractABI
	fc2.NextContractExpectedCaller = goContractAddr2
	require.NoError(t, callGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckTxCaller", &testdata.CallArgs{}))
	require.NoError(t, staticCallGoContractMethod(vm, vmAddr1, goContractAddr1, "CheckQueryCaller", &testdata.StaticCallArgs{}))
}

func TestGetEvmTxReceipt(t *testing.T) {
	evmAuxStore, err := rcommon.NewMockEvmAuxStore()
	require.NoError(t, err)
	createRegistry, err := registry.NewRegistryFactory(registry.LatestRegistryVersion)
	require.NoError(t, err)
	receiptHandler := handler.NewReceiptHandler(
		loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher()),
		handler.DefaultMaxReceipts,
		evmAuxStore,
	)
	require.NoError(t, err)

	state := rcommon.MockState(1)
	txHash := mockTxHash(1)
	err = receiptHandler.CacheReceipt(state, vmAddr1, vmAddr2, []*ptypes.EventData{}, nil, txHash)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(1))

	state20 := rcommon.MockStateAt(state, 20)
	vm := NewPluginVM(NewStaticLoader(), state20, createRegistry(state20), &fakeEventHandler{}, nil, nil, nil, receiptHandler)
	contractCtx := vm.CreateContractContext(vmAddr1, vmAddr2, true)
	receipt, err := contractCtx.GetEvmTxReceipt(txHash)
	require.NoError(t, err)
	require.EqualValues(t, 0, bytes.Compare(txHash, receipt.TxHash))
	require.EqualValues(t, 0, bytes.Compare(vmAddr2.Local, receipt.ContractAddress))
	require.EqualValues(t, int64(1), receipt.BlockNumber)
	require.NoError(t, evmAuxStore.Close())
}

//This test should handle the case of pending transactions being readable
func TestGetEvmTxReceiptNoCommit(t *testing.T) {
	evmAuxStore, err := rcommon.NewMockEvmAuxStore()
	require.NoError(t, err)
	createRegistry, err := registry.NewRegistryFactory(registry.LatestRegistryVersion)
	require.NoError(t, err)
	receiptHandler := handler.NewReceiptHandler(
		loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher()),
		handler.DefaultMaxReceipts,
		evmAuxStore,
	)

	state := rcommon.MockState(1)
	txHash := mockTxHash(1)
	err = receiptHandler.CacheReceipt(state, vmAddr1, vmAddr2, []*ptypes.EventData{}, nil, txHash)
	require.NoError(t, err)

	state20 := rcommon.MockStateAt(state, 20)
	vm := NewPluginVM(NewStaticLoader(), state20, createRegistry(state20), &fakeEventHandler{}, nil, nil, nil, receiptHandler)
	contractCtx := vm.CreateContractContext(vmAddr1, vmAddr2, true)
	receipt, err := contractCtx.GetEvmTxReceipt(txHash)
	require.NoError(t, err)
	require.EqualValues(t, 0, bytes.Compare(txHash, receipt.TxHash))
	require.EqualValues(t, 0, bytes.Compare(vmAddr2.Local, receipt.ContractAddress))
	require.EqualValues(t, int64(1), receipt.BlockNumber)
	require.NoError(t, evmAuxStore.Close())
}

func deployGoContract(vm *PluginVM, contractID string, contractNum uint64, owner loom.Address) (loom.Address, error) {
	init, err := proto.Marshal(&Request{
		ContentType: loom_plugin.EncodingType_PROTOBUF3,
		Accept:      loom_plugin.EncodingType_PROTOBUF3,
		Body:        nil,
	})
	if err != nil {
		return loom.Address{}, err
	}
	pc := &PluginCode{
		Name:  contractID,
		Input: init,
	}
	code, err := proto.Marshal(pc)
	if err != nil {
		return loom.Address{}, err
	}
	callerAddr := CreateAddress(owner, contractNum)
	_, contractAddr, err := vm.Create(callerAddr, code, loom.NewBigUIntFromInt(0))
	if err != nil {
		return loom.Address{}, err
	}
	return contractAddr, nil
}

func encodeGoCallInput(method string, inpb proto.Message) ([]byte, error) {
	args, err := proto.Marshal(inpb)
	if err != nil {
		return nil, err
	}

	body, err := proto.Marshal(&loom_plugin.ContractMethodCall{
		Method: method,
		Args:   args,
	})
	if err != nil {
		return nil, err
	}

	input, err := proto.Marshal(&loom_plugin.Request{
		ContentType: loom_plugin.EncodingType_PROTOBUF3,
		Accept:      loom_plugin.EncodingType_PROTOBUF3,
		Body:        body,
	})
	if err != nil {
		return nil, err
	}

	return input, nil
}

func callGoContractMethod(vm *PluginVM, callerAddr, contractAddr loom.Address, method string, inpb proto.Message) error {
	input, err := encodeGoCallInput(method, inpb)
	if err != nil {
		return err
	}
	_, err = vm.Call(callerAddr, contractAddr, input, loom.NewBigUIntFromInt(0))
	return err
}

func staticCallGoContractMethod(vm *PluginVM, callerAddr, contractAddr loom.Address, method string, inpb proto.Message) error {
	input, err := encodeGoCallInput(method, inpb)
	if err != nil {
		return err
	}
	_, err = vm.StaticCall(callerAddr, contractAddr, input)
	return err
}

func deployEVMContract(vm lvm.VM, filename string, caller loom.Address) (loom.Address, *abi.ABI, error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	if err != nil {
		return contractAddr, nil, err
	}
	abiBytes, err := ioutil.ReadFile("testdata/" + filename + ".abi")
	if err != nil {
		return contractAddr, nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return contractAddr, nil, err
	}
	byteCode := common.FromHex(string(hexByteCode))
	_, contractAddr, err = vm.Create(caller, byteCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return contractAddr, nil, err
	}
	return contractAddr, &contractABI, nil
}

func callEVMContractMethod(ctx contract.Context, contractAddr loom.Address, contractABI *abi.ABI,
	method string, params ...interface{}) error {
	input, err := contractABI.Pack(method, params...)
	if err != nil {
		return err
	}
	// TODO: this shouldn't panic if the output is nil!
	output := []byte{}
	return contract.CallEVM(ctx, contractAddr, input, &output)
}

func staticCallEVMContractMethod(ctx contract.StaticContext, contractAddr loom.Address, contractABI *abi.ABI,
	method string, result interface{}) error {
	input, err := contractABI.Pack(method)
	if err != nil {
		return err
	}
	output := []byte{}
	err = contract.StaticCallEVM(ctx, contractAddr, input, &output)
	if err != nil {
		return err
	}
	return contractABI.Unpack(result, method, output)
}

func mockTxHash(nonce uint64) []byte {
	tx := gtypes.NewContractCreation(
		nonce, nil, math.MaxUint64, big.NewInt(0), []byte(""),
	)
	return tx.Hash().Bytes()
}
