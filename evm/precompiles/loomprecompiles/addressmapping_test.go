package loomprecompiles

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/log"
	lplugin "github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
)

const (
	BlockHeight = int64(34)
)

const (
	mainChain = "default"
	chain2    = "loom"
)

var (
	blockTime = time.Unix(123456789, 0)
	caller    = loom.MustParseAddress(mainChain + ":0xb16a379ec18d4093666f8f38b11a3071c920207d")
	altCaller = loom.MustParseAddress(chain2 + ":0xb16a379ec18d4093666f8f38b11a3071c920207d")
)

func TestMapAccount(t *testing.T) {
	mockState := mockState(mainChain)
	manager := vm.NewManager()
	manager.Register(vm.VMType_EVM, loomVmFactory(manager))
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(mockState)

	loader := lplugin.NewStaticLoader(address_mapper.Contract)
	manager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return lplugin.NewPluginVM(
			loader,
			state,
			reg,
			nil, log.Default, nil, nil, nil), nil
	})

	mockState.SetFeature(features.EvmConstantinopleFeature, true)
	evmVm, err := manager.InitVM(vm.VMType_EVM, mockState)
	require.NoError(t, err)
	pvm, err := manager.InitVM(vm.VMType_PLUGIN, mockState)
	require.NoError(t, err)

	amCode, err := json.Marshal(address_mapper.InitRequest{})
	require.NoError(t, err)
	amInitCode, err := loadContractCode("addressmapper:0.1.0", amCode)
	require.NoError(t, err)
	callerAddr := lplugin.CreateAddress(loom.RootAddress("chain"), uint64(1))
	_, amAddr, err := pvm.Create(callerAddr, amInitCode, loom.NewBigUIntFromInt(0))
	require.NoError(t, err)
	err = reg.Register("addressmapper", amAddr, amAddr)
	require.NoError(t, err)

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	ethAddr := loom.Address{ChainID: "eth", Local: local}

	ctx, err := newContractContext(
		"addressmapper",
		pvm.(*lplugin.PluginVM),
		altCaller,
		false,
	)

	require.NoError(t, err)

	amContract := &address_mapper.AddressMapper{}
	require.NoError(t, amContract.Init(ctx, &address_mapper.InitRequest{}))

	sig, err := address_mapper.SignIdentityMapping(ethAddr, altCaller, privateKey, evmcompat.SignatureType_EIP712)
	require.NoError(t, err)
	require.NoError(t, amContract.AddIdentityMapping(ctx, &address_mapper.AddIdentityMappingRequest{
		From:      ethAddr.MarshalPB(),
		To:        altCaller.MarshalPB(),
		Signature: sig,
	}))
	abiPc, pcAddr := deploySolContract(t, caller, "TestLoomNativeApi", evmVm)

	signer := &auth.EthSigner66Byte{PrivateKey: privateKey}
	mockTx := []byte("message")
	signature := signer.Sign(mockTx)
	var hash, r, s [32]byte
	copy(hash[:], sha3.SoliditySHA3(mockTx)[:])
	copy(r[:], signature[1:33])
	copy(s[:], signature[33:65])
	v := signature[65]

	input, err := abiPc.Pack("TestMappedAccount", "eth", hash, v, r, s, chain2)
	require.NoError(t, err, "packing parameters")

	ret, err := evmVm.StaticCall(caller, pcAddr, input)
	require.NoError(t, err)
	require.Len(t, ret, 32)
	require.Equal(t, 0, bytes.Compare(altCaller.Local, ret[len(ret)-20:]))
}

func TestMapToLoomAccount(t *testing.T) {
	mockState := mockState(mainChain)
	manager := vm.NewManager()
	manager.Register(vm.VMType_EVM, loomVmFactory(manager))
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(mockState)

	loader := lplugin.NewStaticLoader(address_mapper.Contract)
	manager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return lplugin.NewPluginVM(
			loader,
			state,
			reg,
			nil, log.Default, nil, nil, nil), nil
	})

	mockState.SetFeature(features.EvmConstantinopleFeature, true)
	evmVm, err := manager.InitVM(vm.VMType_EVM, mockState)
	require.NoError(t, err)
	pvm, err := manager.InitVM(vm.VMType_PLUGIN, mockState)
	require.NoError(t, err)

	amCode, err := json.Marshal(address_mapper.InitRequest{})
	require.NoError(t, err)
	amInitCode, err := loadContractCode("addressmapper:0.1.0", amCode)
	require.NoError(t, err)
	callerAddr := lplugin.CreateAddress(loom.RootAddress("chain"), uint64(1))
	_, amAddr, err := pvm.Create(callerAddr, amInitCode, loom.NewBigUIntFromInt(0))
	require.NoError(t, err)
	err = reg.Register("addressmapper", amAddr, amAddr)
	require.NoError(t, err)

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	ethAddr := loom.Address{ChainID: "eth", Local: local}

	ctx, err := newContractContext(
		"addressmapper",
		pvm.(*lplugin.PluginVM),
		caller,
		false,
	)

	require.NoError(t, err)

	amContract := &address_mapper.AddressMapper{}
	require.NoError(t, amContract.Init(ctx, &address_mapper.InitRequest{}))

	sig, err := address_mapper.SignIdentityMapping(ethAddr, caller, privateKey, evmcompat.SignatureType_EIP712)
	require.NoError(t, err)
	require.NoError(t, amContract.AddIdentityMapping(ctx, &address_mapper.AddIdentityMappingRequest{
		From:      ethAddr.MarshalPB(),
		To:        caller.MarshalPB(),
		Signature: sig,
	}))
	abiPc, pcAddr := deploySolContract(t, caller, "TestLoomNativeApi", evmVm)

	signer := &auth.EthSigner66Byte{PrivateKey: privateKey}
	mockTx := []byte("message")
	signature := signer.Sign(mockTx)
	var hash, r, s [32]byte
	copy(hash[:], sha3.SoliditySHA3(mockTx)[:])
	copy(r[:], signature[1:33])
	copy(s[:], signature[33:65])
	v := signature[65]

	input, err := abiPc.Pack("TestMappedLoomAccount", "eth", hash, v, r, s)
	require.NoError(t, err, "packing parameters")

	ret, err := evmVm.StaticCall(caller, pcAddr, input)
	require.NoError(t, err)
	require.Len(t, ret, 32)
	require.Equal(t, 0, bytes.Compare(caller.Local, ret[len(ret)-20:]))
}

func loomVmFactory(vmManager *vm.Manager) func(state loomchain.State) (vm.VM, error) {
	return func(state loomchain.State) (vm.VM, error) {
		eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())
		receiptHandlerProvider := receipts.NewReceiptHandlerProvider(
			eventHandler,
			handler.DefaultMaxReceipts,
			nil,
		)
		receiptHandler := receiptHandlerProvider.Writer()
		return evm.NewLoomVm(
			state,
			eventHandler,
			receiptHandler,
			nil,
			NewLoomPrecompileHandler(getContractStaticCtx("addressmapper", vmManager)),
			false,
		), nil
	}
}

func getContractStaticCtx(pluginName string, vmManager *vm.Manager) func(state loomchain.State) (contractpb.StaticContext, error) {
	return func(state loomchain.State) (contractpb.StaticContext, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
		if err != nil {
			return nil, err
		}
		return lplugin.NewInternalContractContext(pluginName, pvm.(*lplugin.PluginVM), true)
	}
}

func newContractContext(contractName string, pluginVM *lplugin.PluginVM, caller loom.Address, readOnly bool) (contractpb.Context, error) {
	contractAddr, err := pluginVM.Registry.Resolve(contractName)
	if err != nil {
		return nil, err
	}
	return contractpb.WrapPluginContext(pluginVM.CreateContractContext(caller, contractAddr, readOnly)), nil
}

func mockState(chainID string) loomchain.State {
	header := abci.Header{}
	header.Height = BlockHeight
	header.Time = blockTime
	header.ChainID = chainID
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header, nil, nil)
}

func deploySolContract(t *testing.T, caller loom.Address, filename string, vm vm.VM) (abi.ABI, loom.Address) {
	bytetext, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	require.NoError(t, err, "reading "+filename+".bin")
	str := string(bytetext)
	_ = str
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")

	_, addr, err := vm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))

	require.NoError(t, err, "deploying "+filename+" on EVM")
	simpleStoreData, err := ioutil.ReadFile("testdata/" + filename + ".abi")
	require.NoError(t, err, "reading "+filename+".abi")
	ethAbi, err := abi.JSON(strings.NewReader(string(simpleStoreData)))
	require.NoError(t, err, "reading abi")
	return ethAbi, addr
}

func loadContractCode(location string, init json.RawMessage) ([]byte, error) {
	body, err := init.MarshalJSON()
	if err != nil {
		return nil, err
	}

	req := &plugin.Request{
		ContentType: plugin.EncodingType_JSON,
		Body:        body,
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	pluginCode := &lplugin.PluginCode{
		Name:  location,
		Input: input,
	}
	return proto.Marshal(pluginCode)
}
