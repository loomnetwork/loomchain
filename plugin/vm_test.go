// +build evm

package plugin

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	proto "github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	loom_plugin "github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/testdata"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/subs"
	levm "github.com/loomnetwork/loomchain/evm"
	registry "github.com/loomnetwork/loomchain/registry"
	regFactory "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	vmAddr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	vmAddr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	vmAddr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

// Implements loomchain.EventHandler interface
type fakeEventHandler struct {
}

func (eh *fakeEventHandler) Post(height uint64, e *loomchain.EventData) error {
	return nil
}

func (eh *fakeEventHandler) EmitBlockTx(height uint64) error {
	return nil
}

func (eh *fakeEventHandler) SubscriptionSet() *loomchain.SubscriptionSet {
	return nil
}

func (eh *fakeEventHandler) EthSubscriptionSet() *subs.EthSubscriptionSet {
	return nil
}

type VMTestContract struct {
	t                          *testing.T
	Name                       string
	Version                    string
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
		Version: c.Version,
	}, nil
}

func (c *VMTestContract) Init(ctx contract.Context, req *loom_plugin.Request) error {
	return nil
}

func (c *VMTestContract) GetVersion(ctx contract.StaticContext, req *loom_plugin.Request) (*testdata.StaticCallResult, error) {
	return &testdata.StaticCallResult{Result: c.Version}, nil
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

func TestPluginVMMultipleVersionContract(t *testing.T) {
	fc1 := &VMTestContract{t: t, Name: "fakecontract", Version: "0.0.1"}
	fc2 := &VMTestContract{t: t, Name: "fakecontract", Version: "0.0.2"}
	fc3 := &VMTestContract{t: t, Name: "fakecontract", Version: "0.0.3"}

	loader := NewStaticLoader(
		contract.MakePluginContract(fc1),
		contract.MakePluginContract(fc2),
		contract.MakePluginContract(fc3),
	)

	block := abci.Header{
		ChainID: "chain",
		Height:  int64(34),
		Time:    int64(123456789),
	}
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), block)
	createRegistry, err := regFactory.NewRegistryFactory(regFactory.LatestRegistryVersion)
	require.NoError(t, err)
	vm := NewPluginVM(loader, state, createRegistry(state), &fakeEventHandler{}, nil, nil)

	owner := loom.RootAddress("chain")
	goContractAddr1, err := deployGoContract(vm, "fakecontract", "0.0.1", 0, owner)
	require.NoError(t, err)
	goContractAddr2, err := deployGoContract(vm, "fakecontract", "0.0.2", 0, owner)
	require.NoError(t, err)
	goContractAddr3, err := deployGoContract(vm, "fakecontract", "0.0.3", 0, owner)
	require.NoError(t, err)

	// Making sure different versions are sharing same address
	assert.Equal(t, goContractAddr1, goContractAddr2)
	assert.Equal(t, goContractAddr2, goContractAddr3)

	input, err := encodeGoCallInput("GetVersion", &testdata.CallArgs{})
	require.NoError(t, err)

	// Correct contract object should be called
	response, err := vm.StaticCall(owner, goContractAddr1, "0.0.1", input)
	output, err := decodeStaticCallResult(response)
	require.NoError(t, err)
	assert.Equal(t, output.Result, "0.0.1")

	response, err = vm.StaticCall(owner, goContractAddr2, "0.0.2", input)
	output, err = decodeStaticCallResult(response)
	require.NoError(t, err)
	assert.Equal(t, output.Result, "0.0.2")

	response, err = vm.StaticCall(owner, goContractAddr3, "0.0.3", input)
	output, err = decodeStaticCallResult(response)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(output.Result), "0.0.3")

}

func TestPluginVMContractContextCaller(t *testing.T) {
	fc1 := &VMTestContract{t: t, Name: "fakecontract1", Version: "0.0.1"}
	fc2 := &VMTestContract{t: t, Name: "fakecontract2", Version: "0.0.1"}
	fc3 := &VMTestContract{t: t, Name: "fakecontract3", Version: "0.0.1"}

	loader := NewStaticLoader(
		contract.MakePluginContract(fc1),
		contract.MakePluginContract(fc2),
		contract.MakePluginContract(fc3),
	)
	block := abci.Header{
		ChainID: "chain",
		Height:  int64(34),
		Time:    int64(123456789),
	}
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), block)
	createRegistry, err := regFactory.NewRegistryFactory(regFactory.LatestRegistryVersion)
	require.NoError(t, err)
	vm := NewPluginVM(loader, state, createRegistry(state), &fakeEventHandler{}, nil, nil, nil)
	evm := levm.NewLoomVm(state, nil, nil, nil)

	// Deploy contracts
	owner := loom.RootAddress("chain")
	goContractAddr1, err := deployGoContract(vm, "fakecontract1:0.0.1", registry.DefaultContractVersion, 0, owner)
	require.NoError(t, err)
	goContractAddr2, err := deployGoContract(vm, "fakecontract2:0.0.1", registry.DefaultContractVersion, 1, owner)
	require.NoError(t, err)
	goContractAddr3, err := deployGoContract(vm, "fakecontract3:0.0.1", registry.DefaultContractVersion, 2, owner)
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

func deployGoContract(vm *PluginVM, contractID string, contractVersion string, contractNum uint64, owner loom.Address) (loom.Address, error) {
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
	_, contractAddr, err := vm.Create(callerAddr, contractVersion, code, loom.NewBigUIntFromInt(0))
	if err != nil {
		return loom.Address{}, err
	}
	return contractAddr, nil
}

func decodeStaticCallResult(response []byte) (*testdata.StaticCallResult, error) {
	var output loom_plugin.Response
	var result testdata.StaticCallResult
	err := proto.Unmarshal(response, &output)
	err = proto.Unmarshal(output.Body, &result)
	return &result, err
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
	_, err = vm.Call(callerAddr, contractAddr, registry.DefaultContractVersion, input, loom.NewBigUIntFromInt(0))
	return err
}

func staticCallGoContractMethod(vm *PluginVM, callerAddr, contractAddr loom.Address, method string, inpb proto.Message) error {
	input, err := encodeGoCallInput(method, inpb)
	if err != nil {
		return err
	}
	_, err = vm.StaticCall(callerAddr, contractAddr, registry.DefaultContractVersion, input)
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
	_, contractAddr, err = vm.Create(caller, registry.DefaultContractVersion, byteCode, loom.NewBigUIntFromInt(0))
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
