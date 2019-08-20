// +build evm

package sample_go_contract

import (
	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/db"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	addr1  = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	caller = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestSampleGoContract(t *testing.T) {
	testingInit := types.SampleGoContractInitRequest{}

	state, reg, manager, err := karma.MockStateWithContracts(
		db.NewMemDB(),
		karma.MockContractDetails{SampleGoCongress, "1.0.0", testingInit, Contract},
	)
	require.NoError(t, err)
	//var eventDispatcher loomchain.EventDispatcher
	//var eventHandler loomchain.EventHandler = loomchain.NewDefaultEventHandler(eventDispatcher)

	require.NoError(t, err)
	pluginVm, err := manager.InitVM(vm.VMType_PLUGIN, state)
	require.NoError(t, err)

	addr, err := reg.Resolve(SampleGoCongress)
	require.NoError(t, err)
	ctx := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr1, addr, pluginVm),
	)
	testingContract := &SampleGoContract{}

	manager.Register(vm.VMType_EVM, func(state loomchain.State) (vm.VM, error) {
		return evm.NewLoomVm(state, nil, nil, nil, false), nil
	})
	pluginEvm, err := manager.InitVM(vm.VMType_EVM, state)
	require.NoError(t, err)

	bytetext, err := ioutil.ReadFile("testdata/TestEvent.bin")
	require.NoError(t, err)
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")

	_, testEventAddr, err := pluginEvm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))
	require.NoError(t, err)

	bytetext, err = ioutil.ReadFile("testdata/ChainTestEvent.bin")
	require.NoError(t, err)
	bytecode, err = hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")
	_, testChainEventAddr, err := pluginEvm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))
	require.NoError(t, err)

	require.NoError(t, testingContract.TestNestedEvmCalls(ctx, &types.SampleGoContractNestedEvmRequest{}))
	require.NoError(t, err)

	req := types.SampleGoContractNestedEvm2Request{
		TestEvent:      testEventAddr.MarshalPB(),
		ChainTestEvent: testChainEventAddr.MarshalPB(),
	}
	require.NoError(t, testingContract.TestNestedEvmCalls2(ctx, &req))
}
