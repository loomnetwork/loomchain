// +build evm

package sample_go_contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	types "github.com/loomnetwork/go-loom/builtin/types/testing"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

const (
	testEventAbi      = `[{"constant":false,"inputs":[{"name":"i","type":"uint256"}],"name":"sendEvent","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"number","type":"uint256"}],"name":"MyEvent","type":"event"}]`
	chainTestEventAbi = `[{"constant":false,"inputs":[{"name":"i","type":"uint256"}],"name":"chainEvent","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
)

func (k *SampleGoContract) TestNestedEvmCalls(ctx contractpb.Context, req *types.TestingNestedEvmRequest) error {
	if err := testEventCall(ctx); err != nil {
		return err
	}
	if err := testChainEventCall(ctx); err != nil {
		return err
	}
	return nil
}

func testEventCall(ctx contractpb.Context) error {
	testEventAddr, err := ctx.Resolve("TestEvent")
	abiEventData, err := abi.JSON(strings.NewReader(testEventAbi))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("sendEvent", big.NewInt(65))
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

func testChainEventCall(ctx contractpb.Context) error {
	testEventAddr, err := ctx.Resolve("ChainTestEvent")
	abiEventData, err := abi.JSON(strings.NewReader(chainTestEventAbi))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("chainEvent", big.NewInt(33))
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
