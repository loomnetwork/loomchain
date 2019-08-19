// +build evm

package sample_go_contract

import (
	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/testing"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/db"

	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	addr1  = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	caller = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestSampleGoContract(t *testing.T) {
	testingInit := types.TestingInitRequest{}

	state, reg, manager, err := karma.MockStateWithContracts(
		db.NewMemDB(),
		karma.MockContractDetails{"testing", "1.0.0", testingInit, Contract},
	)
	require.NoError(t, err)
	pluginVm, err := manager.InitVM(vm.VMType_PLUGIN, state)

	addr, err := reg.Resolve("testing")
	require.NoError(t, err)
	ctx := contractpb.WrapPluginContext(
		karma.CreateFakeStateContext(state, reg, addr1, addr, pluginVm),
	)
	testingContract := &SampleGoContract{}

	bytetext, err := ioutil.ReadFile("testdata/TestEvent.bin")
	require.NoError(t, err)
	bytecode, err := hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")
	_, _, err = pluginVm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))

	bytetext, err = ioutil.ReadFile("testdata/ChainTestEvent.bin")
	require.NoError(t, err)
	bytecode, err = hex.DecodeString(string(bytetext))
	require.NoError(t, err, "decoding bytecode")
	_, addr, err = pluginVm.Create(caller, bytecode, loom.NewBigUIntFromInt(0))

	require.NoError(t, testingContract.TestNestedEvmCalls(ctx, &types.TestingNestedEvmRequest{}))

}
