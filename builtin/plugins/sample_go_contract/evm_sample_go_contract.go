// +build evm

package sample_go_contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

const (
	testEventAbi      = `[{"constant":false,"inputs":[{"name":"i","type":"uint256"}],"name":"sendEvent","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"number","type":"uint256"}],"name":"MyEvent","type":"event"}]`
	chainTestEventAbi = `[{"constant":false,"inputs":[{"name":"i","type":"uint256"}],"name":"chainEvent","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
)

func (k *SampleGoContract) TestNestedEvmCalls(ctx contractpb.Context, req *types.SampleGoContractNestedEvmRequest) error {
	testEventAddr, err := ctx.Resolve("TestEvent")
	if err != nil {
		return nil
	}
	if err := testEventCall(ctx, testEventAddr, 65); err != nil {
		return err
	}
	testChainEventAddr, err := ctx.Resolve("ChainTestEvent")
	if err != nil {
		return nil
	}
	if err := testChainEventCall(ctx, testChainEventAddr, 33); err != nil {
		return err
	}
	return nil
}

func (k *SampleGoContract) TestNestedEvmCalls2(ctx contractpb.Context, req *types.SampleGoContractNestedEvm2Request) error {
	if err := testEventCall(ctx, loom.UnmarshalAddressPB(req.TestEvent), req.TestEventValue); err != nil {
		return err
	}
	if err := testChainEventCall(ctx, loom.UnmarshalAddressPB(req.ChainTestEvent), req.ChainTestEventValue); err != nil {
		return err
	}
	return nil
}

func testEventCall(ctx contractpb.Context, testEventAddr loom.Address, value uint64) error {
	abiEventData, err := abi.JSON(strings.NewReader(testEventAbi))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("sendEvent", big.NewInt(int64(value)))
	if err != nil {
		return err
	}
	evmOut := []byte{}
	err = contractpb.CallEVM(ctx, testEventAddr, input, &evmOut)
	if err != nil {
		return err
	}
	return nil
}

func testChainEventCall(ctx contractpb.Context, testChainEventAddr loom.Address, value uint64) error {
	abiEventData, err := abi.JSON(strings.NewReader(chainTestEventAbi))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("chainEvent", big.NewInt(int64(value)))
	if err != nil {
		return err
	}
	evmOut := []byte{}
	err = contractpb.CallEVM(ctx, testChainEventAddr, input, &evmOut)
	if err != nil {
		return err
	}
	return nil
}
