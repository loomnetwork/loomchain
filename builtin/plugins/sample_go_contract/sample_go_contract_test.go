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

	"github.com/ethereum/go-ethereum/common"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/plugin"
)

var (
	addr1  = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	caller = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestSampleGoContract(t *testing.T) {
	pctx := plugin.CreateFakeContextWithEVM(caller, addr1)

	sampleGoContract := &SampleGoContract{}
	sampleAddr := pctx.CreateContract(Contract)
	ctx := contractpb.WrapPluginContext(pctx.WithAddress(sampleAddr))
	sampleInit := types.SampleGoContractInitRequest{}
	require.NoError(t, sampleGoContract.Init(ctx, &sampleInit))

	pctx.State.SetFeature(loomchain.EvmConstantinopleFeature, true)

	testEventaddr, err := deployContractToEVM(pctx, "TestEvent", caller)
	require.NoError(t, err)

	testChainEventAddr, err := deployContractToEVM(pctx, "ChainTestEvent", caller)
	require.NoError(t, err)

	require.NoError(t, sampleGoContract.TestNestedEvmCalls(ctx, &types.SampleGoContractNestedEvmRequest{}))
	require.NoError(t, err)

	req := types.SampleGoContractNestedEvm2Request{
		TestEvent:      testEventaddr.MarshalPB(),
		ChainTestEvent: testChainEventAddr.MarshalPB(),
	}
	require.NoError(t, sampleGoContract.TestNestedEvmCalls2(ctx, &req))
}

func deployContractToEVM(ctx *plugin.FakeContextWithEVM, filename string, caller loom.Address) (loom.Address, error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	if err != nil {
		return contractAddr, err
	}
	byteCode := common.FromHex(string(hexByteCode))
	byteCode, err = hex.DecodeString(string(hexByteCode))

	vm := evm.NewLoomVm(ctx.State, nil, nil, nil, false)
	_, contractAddr, err = vm.Create(caller, byteCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return contractAddr, err
	}

	ctx.RegisterContract("", contractAddr, caller)
	return contractAddr, nil
}
